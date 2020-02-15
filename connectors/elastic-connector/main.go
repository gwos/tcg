package main

import (
	"github.com/gwos/tng/log"
)

func main() {
	monitoredResources := CollectMetrics()
	log.Info(len(monitoredResources))
}
