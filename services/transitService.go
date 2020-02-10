package services

import (
	"github.com/gwos/tng/nats"
	"sync"
)

// TransitService implements AgentServices, TransitServices interfaces
type TransitService struct {
	*AgentService
}

var onceTransitService sync.Once
var transitService *TransitService

// GetTransitService implements Singleton pattern
func GetTransitService() *TransitService {
	onceTransitService.Do(func() {
		transitService = &TransitService{GetAgentService()}
	})
	return transitService
}

// SendResourceWithMetrics implements TransitServices.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(payload []byte) error {
	res, err := service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	return nats.Publish(SubjSendResourceWithMetrics, res)
}

// SynchronizeInventory implements TransitServices.SynchronizeInventory interface
func (service *TransitService) SynchronizeInventory(payload []byte) error {
	res, err := service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	return nats.Publish(SubjSynchronizeInventory, res)
}
