package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/transit"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// TransitService implements AgentServices, TransitServices interfaces
type TransitService struct {
	*AgentService
	listMetricsHandler func() ([]byte, error)

	batchBuffer struct {
		exit   chan bool
		cache  *cache.Cache
		ticker *time.Ticker
	}
	batchRequest BatchRequest
}

type BatchRequest struct {
	groupsMap map[string]transit.ResourceGroup
	resMap    map[string]transit.DynamicMonitoredResource
	resSvcMap map[string][]transit.DynamicMonitoredService
	svcMap    map[string]svcMapItem

	r    transit.DynamicResourcesWithServicesRequest
	size int
}

// Fill takes transit.DynamicResourcesWithServicesRequest as RawJSON
func (br *BatchRequest) Fill(b []byte) {
	if br.groupsMap == nil {
		br.groupsMap = make(map[string]transit.ResourceGroup)
	}
	if br.resMap == nil {
		br.resMap = make(map[string]transit.DynamicMonitoredResource)
	}
	if br.resSvcMap == nil {
		br.resSvcMap = make(map[string][]transit.DynamicMonitoredService)
	}
	if br.svcMap == nil {
		br.svcMap = make(map[string]svcMapItem)
	}

	var p transit.DynamicResourcesWithServicesRequest
	if err := json.Unmarshal(b, &p); err != nil {
		log.Err(err).
			RawJSON("payload", b).
			Msg("could not unmarshal metrics payload for batch")
	} else {
		if br.r.Context == nil {
			br.r.Context = p.Context
		}
		for _, g := range p.Groups {
			br.groupsMap[string(g.Type)+":"+g.GroupName] = g
		}
		for _, res := range p.Resources {
			resK := string(res.Type) + ":" + res.Name
			for _, svc := range res.Services {
				svcK := string(res.Type) + ":" + res.Name + "::" + string(svc.Type) + ":" + svc.Name
				br.svcMap[svcK] = svcMapItem{
					resK: resK,
					svcK: svcK,
					svc:  svc,
				}
			}
			res.Services = []transit.DynamicMonitoredService{}
			br.resMap[resK] = res
		}
		br.size += len(b)
		lim := GetTransitService().BatchMaxBytes
		if br.size > lim {
			log.Debug().Msgf("batch payload exceeded the soft limit %d KB", lim/1024)
			br.Flush()
		}
	}
}

// Flush sends the request if not empty
func (br *BatchRequest) Flush() error {
	defer func() {
		br.groupsMap = make(map[string]transit.ResourceGroup)
		br.resMap = make(map[string]transit.DynamicMonitoredResource)
		br.resSvcMap = make(map[string][]transit.DynamicMonitoredService)
		br.svcMap = make(map[string]svcMapItem)
		br.size = 0
	}()

	for _, g := range br.groupsMap {
		br.r.Groups = append(br.r.Groups, g)
	}
	for _, svcItem := range br.svcMap {
		br.resSvcMap[svcItem.resK] = append(br.resSvcMap[svcItem.resK], svcItem.svc)
	}
	for _, res := range br.resMap {
		resK := string(res.Type) + ":" + res.Name
		res.Services = br.resSvcMap[resK]
		br.r.Resources = append(br.r.Resources, res)
	}

	if len(br.r.Resources) > 0 {
		log.Debug().Msgf("batched %d resources with %d services in %d groups",
			len(br.r.Resources), len(br.svcMap), len(br.groupsMap))
		b, err := json.Marshal(br.r)
		if err != nil {
			return err
		}

		if bytes.Contains(b, []byte(`:"-6`)) {
			log.Warn().
				Interface("resMap", br.resMap).
				Interface("batch.Resources", br.r.Resources).
				RawJSON("payload", b).
				Msg("zero time in batch payload")
		}

		GetTransitService().sendMetrics(context.Background(), b)
	}
	return nil
}

type svcMapItem struct {
	resK string
	svcK string
	svc  transit.DynamicMonitoredService
}

var onceTransitService sync.Once
var transitService *TransitService

// GetTransitService implements Singleton pattern
func GetTransitService() *TransitService {
	onceTransitService.Do(func() {
		transitService = &TransitService{
			AgentService:       GetAgentService(),
			listMetricsHandler: defaultListMetricsHandler,
		}
		/* init the batcher buffer to reduce config logic */
		transitService.batchBuffer.cache = cache.New(-1, -1)
		transitService.batchBuffer.cache.SetDefault("nextCK", uint64(0))
		transitService.batchBuffer.exit = make(chan bool, 1)
		batchMetrics := transitService.Connector.BatchMetrics
		if batchMetrics == 0 {
			batchMetrics = math.MaxInt64
		}
		transitService.batchBuffer.ticker = time.NewTicker(batchMetrics)
		go func() {
			for {
				select {
				case <-transitService.batchBuffer.exit:
					return
				case <-transitService.batchBuffer.ticker.C:
					transitService.batchBufferedMetrics()
				}
			}
		}()
	})
	return transitService
}

func defaultListMetricsHandler() ([]byte, error) {
	return nil, fmt.Errorf("listMetricsHandler unavailable")
}

// ListMetrics implements TransitServices.ListMetrics interface
func (service *TransitService) ListMetrics() ([]byte, error) {
	return service.listMetricsHandler()
}

// RegisterListMetricsHandler implements TransitServices.RegisterListMetricsHandler interface
func (service *TransitService) RegisterListMetricsHandler(handler func() ([]byte, error)) {
	service.listMetricsHandler = handler
}

// RemoveListMetricsHandler implements TransitServices.RemoveListMetricsHandler interface
func (service *TransitService) RemoveListMetricsHandler() {
	service.listMetricsHandler = defaultListMetricsHandler
}

// ClearInDowntime implements TransitServices.ClearInDowntime interface
func (service *TransitService) ClearInDowntime(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "ClearInDowntime")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeClearInDowntime}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjDowntime, b)
	return err
}

// SetInDowntime implements TransitServices.SetInDowntime interface
func (service *TransitService) SetInDowntime(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SetInDowntime")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeSetInDowntime}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjDowntime, b)
	return err
}

// SendEvents implements TransitServices.SendEvents interface
func (service *TransitService) SendEvents(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendEvents")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeEvents}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjEvents, b)
	return err
}

// SendEventsAck implements TransitServices.SendEventsAck interface
func (service *TransitService) SendEventsAck(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendEventsAck")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeEventsAck}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjEvents, b)
	return err
}

// SendEventsUnack implements TransitServices.SendEventsUnack interface
func (service *TransitService) SendEventsUnack(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendEventsUnack")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeEventsUnack}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjEvents, b)
	return err
}

// SendResourceWithMetrics implements TransitServices.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(ctx context.Context, payload []byte) error {
	if service.Connector.BatchMetrics == 0 {
		return service.sendMetrics(ctx, payload)
	}
	return service.bufferMetrics(payload)
}

func (service *TransitService) sendMetrics(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendResourceWithMetrics")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	payload, err = service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	b, err = natsPayload{span.SpanContext(), payload, typeMetrics}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjInventoryMetrics, b)
	return err
}

func (service *TransitService) bufferMetrics(payload []byte) error {
	nextCK, err := service.batchBuffer.cache.IncrementUint64("nextCK", 1)
	if err != nil {
		return err
	}
	if nextCK == math.MaxUint64 {
		nextCK = 0
		service.batchBuffer.cache.SetDefault("nextCK", 0)
	}
	service.batchBuffer.cache.SetDefault(strconv.FormatUint(nextCK, 10), payload)
	return err
}

func (service *TransitService) batchBufferedMetrics() error {
	for ck, c := range service.batchBuffer.cache.Items() {
		if ck == "nextCK" {
			continue
		}
		service.batchRequest.Fill(c.Object.([]byte))
		service.batchBuffer.cache.Delete(ck)
	}
	return service.batchRequest.Flush()
}

// SynchronizeInventory implements TransitServices.SynchronizeInventory interface
func (service *TransitService) SynchronizeInventory(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SynchronizeInventory")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	payload, err = service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	b, err = natsPayload{span.SpanContext(), payload, typeInventory}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjInventoryMetrics, b)
	return err
}
