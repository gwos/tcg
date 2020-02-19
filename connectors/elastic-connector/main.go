package main

import (
	"encoding/json"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"os/exec"
)

var transitService = services.GetTransitService()

func main() {
	err := transitService.StartNats()
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = transitService.StartTransport()
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = transitService.StartController()
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer func() {
		err = transitService.StopNats()
		if err != nil {
			log.Error(err.Error())
		}
		err = transitService.StopTransport()
		if err != nil {
			log.Error(err.Error())
		}
		cmd := exec.Command("rm", "-rf", "src")
		_, err = cmd.Output()
		if err != nil {
			log.Error(err.Error())
		}
		err = transitService.StopController()
		if err != nil {
			log.Error(err.Error())
		}
	}()

	monitoredResources, inventoryResources, resourceGroups := CollectMetrics()
	for _, inventoryResource := range inventoryResources {
		if transitService.Status().Transport != services.Stopped {
			inventoryRequest := transit.InventoryRequest{
				Context:   transitService.MakeTracerContext(),
				Resources: []transit.InventoryResource{inventoryResource},
				Groups:    resourceGroups,
			}

			b, err := json.Marshal(inventoryRequest)
			if err != nil {
				log.Error(err)
			}
			err = transitService.SynchronizeInventory(b)
		}
	}
	for _, monitoredResource := range monitoredResources {
		if transitService.Status().Transport != services.Stopped {
			request := transit.ResourcesWithServicesRequest{
				Context:   transitService.MakeTracerContext(),
				Resources: []transit.MonitoredResource{monitoredResource},
			}
			b, err := json.Marshal(request)
			if err != nil {
				log.Error(err)
			}

			err = transitService.SendResourceWithMetrics(b)
			if err != nil {
				log.Error(err)
			}
		}
	}
}
