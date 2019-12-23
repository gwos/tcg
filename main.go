package main

import (
	"encoding/json"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/serverconnector"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/subseconds"
	"github.com/gwos/tng/transit"
	"os/exec"
	"time"
)

var transitService = services.GetTransitService()

// @title TNG API Documentation
// @version 1.0

// @host localhost:8081
// @BasePath /api/v1
func main() {
	err := transitService.StartNats()
	if err != nil {
		log.Error(err.Error())
		return
	}

	defer func() {
		err = transitService.StopNats()
		if err != nil {
			log.Error(err.Error())
		}
		cmd := exec.Command("rm", "-rf", "src")
		_, err = cmd.Output()
		if err != nil {
			log.Error(err.Error())
		}
	}()

	err = sendInventoryResources(*serverconnector.Synchronize())

	for {
		err := sendMonitoredResources(*serverconnector.CollectMetrics())
		if err != nil {
			log.Error(err.Error())
		}

		serverconnector.LastCheck = subseconds.MillisecondTimestamp{Time: time.Now()}

		time.Sleep(20 * time.Second)
	}
}

func sendInventoryResources(resource transit.InventoryResource) error {
	inventoryRequest := transit.InventoryRequest{
		Context: transit.TracerContext{
			AppType:    "VEMA",
			AgentID:    "3939333393342",
			TraceToken: "token-99e93",
			TimeStamp:  subseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.InventoryResource{resource},
		Groups:    nil,
	}

	b, err := json.Marshal(inventoryRequest)
	if err != nil {
		return err
	}

	err = transitService.SynchronizeInventory(b)

	return err
}

func sendMonitoredResources(resource transit.MonitoredResource) error {
	request := transit.ResourcesWithServicesRequest{
		Context: transit.TracerContext{
			AppType:    "VEMA",
			AgentID:    "3939333393342",
			TraceToken: "token-99e93",
			TimeStamp:  subseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.MonitoredResource{resource},
	}

	b, err := json.Marshal(request)
	if err != nil {
		return err
	}

	return transitService.SendResourceWithMetrics(b)
}