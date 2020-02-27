package main

import (
	"github.com/gwos/tng/connectors"
)

func main() {
	monitoredResources, inventoryResources, resourceGroups := CollectMetrics()
	_ = connectors.Start()
	_ = connectors.SendInventory(inventoryResources, resourceGroups)
	_ = connectors.SendMetrics(monitoredResources)
}
