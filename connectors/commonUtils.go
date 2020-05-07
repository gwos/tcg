package connectors

import "github.com/gwos/tng/transit"

func CalcMsSleepAfterSendingInventory(inventory []transit.InventoryResource, max int) int {
	inventorySize := 0
	if inventory != nil && len(inventory) != 0 {
		for _, i := range inventory {
			if i.Services != nil {
				inventorySize = inventorySize + len(i.Services)
			}
		}
	}
	if inventorySize*20 < max {
		return inventorySize * 20
	}
	return max
}
