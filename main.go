package main

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/serverconnector"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"log"
	"os/exec"
	"time"
)

var transitService = services.GetTransitService()

func main() {
	err := transitService.StartNats()
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}

	defer func() {
		err = transitService.StopNats()
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
		cmd := exec.Command("rm", "-rf", "src")
		_, err = cmd.Output()
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
	}()

	err = sendInventoryResources(*serverconnector.Synchronize())

	for {
		err := sendMonitoredResources(*serverconnector.CollectMetrics())
		if err != nil {
			log.Println(err.Error())
		}

		serverconnector.LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}

		time.Sleep(20 * time.Second)
	}
}

func sendInventoryResources(resource transit.InventoryResource) error {
	inventoryRequest := transit.InventoryRequest{
		Context: transit.TracerContext{
			AppType:    "VEMA",
			AgentID:    "3939333393342",
			TraceToken: "token-99e93",
			TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.InventoryResource{resource},
		Groups:    nil,
	}

	b, err := json.Marshal(inventoryRequest)
	if err != nil {
		return err
	}

	return transitService.SynchronizeInventory(b)
}

func sendMonitoredResources(resource transit.MonitoredResource) error {
	request := transit.ResourcesWithServicesRequest{
		Context: transit.TracerContext{
			AppType:    "VEMA",
			AgentID:    "3939333393342",
			TraceToken: "token-99e93",
			TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.MonitoredResource{resource},
	}

	b, err := json.Marshal(request)
	if err != nil {
		return err
	}

	return transitService.SendResourceWithMetrics(b)
}
