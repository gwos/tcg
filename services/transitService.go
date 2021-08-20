package services

import (
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

	batcher struct {
		cache  *cache.Cache
		ticker *time.Ticker
		done   chan bool
	}
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
		/* init the batcher to reduce config logic */
		transitService.batcher.cache = cache.New(-1, -1)
		transitService.batcher.cache.SetDefault("nextCK", uint64(0))
		transitService.batcher.done = make(chan bool, 1)
		batchMetrics := transitService.Connector.BatchMetrics
		if batchMetrics == 0 {
			batchMetrics = math.MaxInt64
		}
		transitService.batcher.ticker = time.NewTicker(batchMetrics)
		go func() {
			for {
				select {
				case <-transitService.batcher.done:
					return
				case <-transitService.batcher.ticker.C:
					transitService.batchCachedMetrics()
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
	return service.cacheMetrics(payload)
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

func (service *TransitService) cacheMetrics(payload []byte) error {
	nextCK, err := service.batcher.cache.IncrementUint64("nextCK", 1)
	if err != nil {
		return err
	}
	if nextCK == math.MaxUint64 {
		nextCK = 0
		service.batcher.cache.SetDefault("nextCK", 0)
	}
	service.batcher.cache.SetDefault(strconv.FormatUint(nextCK, 10), payload)
	return err
}

func (service *TransitService) batchCachedMetrics() error {
	type svcMapItem struct {
		resK string
		svcK string
		svc  transit.DynamicMonitoredService
	}
	batch := transit.DynamicResourcesWithServicesRequest{}
	groupsMap := map[string]transit.ResourceGroup{}
	resMap := map[string]transit.DynamicMonitoredResource{}
	svcMap := map[string]svcMapItem{}
	resSvcMap := map[string][]transit.DynamicMonitoredService{}
	for ck, c := range service.batcher.cache.Items() {
		if ck == "nextCK" {
			continue
		}
		var p transit.DynamicResourcesWithServicesRequest
		if err := json.Unmarshal(c.Object.([]byte), &p); err != nil {
			log.Err(err).
				RawJSON("payload", c.Object.([]byte)).
				Msg("could not unmarshal metrics payload for batch")
		} else {
			log.Debug().
				Interface("p", p).
				RawJSON("payload", c.Object.([]byte)).
				Msg("unmarshal metrics payload for batch")

			if batch.Context == nil {
				batch.Context = p.Context
			}
			for _, g := range p.Groups {
				groupsMap[string(g.Type)+":"+g.GroupName] = g
			}
			for _, res := range p.Resources {
				resK := string(res.Type) + ":" + res.Name
				for _, svc := range res.Services {
					svcK := string(res.Type) + ":" + res.Name + "::" + string(svc.Type) + ":" + svc.Name
					svcMap[svcK] = svcMapItem{
						resK: resK,
						svcK: svcK,
						svc:  svc,
					}
				}
				res.Services = []transit.DynamicMonitoredService{}
				resMap[resK] = res
			}
		}
		service.batcher.cache.Delete(ck)
	}
	for _, g := range groupsMap {
		batch.Groups = append(batch.Groups, g)
	}
	for _, svcItem := range svcMap {
		resSvcMap[svcItem.resK] = append(resSvcMap[svcItem.resK], svcItem.svc)
	}
	for _, res := range resMap {
		resK := string(res.Type) + ":" + res.Name
		res.Services = resSvcMap[resK]
		batch.Resources = append(batch.Resources, res)
	}
	if len(batch.Resources) > 0 {
		log.Debug().Msgf("batched %d resources with %d services in %d groups",
			len(batch.Resources), len(svcMap), len(groupsMap))
		b, err := json.Marshal(batch)
		if err != nil {
			return err
		}
		service.sendMetrics(context.Background(), b)
	}
	return nil
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
