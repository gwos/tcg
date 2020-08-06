package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"net/http"
	"regexp"
	"strings"
)

func initializeEntrypoints() []services.Entrypoint {
	var entrypoints []services.Entrypoint

	entrypoints = append(entrypoints,
		services.Entrypoint{
			Url:     "/receiver",
			Method:  "Post",
			Handler: receiverHandler,
		},
	)

	return entrypoints
}

func receiverHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	if err := processMetrics(body); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
}

func processMetrics(body []byte) error {
	if _, err := parseBody(body); err == nil {
		// fmt.Println(monitoredResource)
		return nil
	} else {
		return err
	}
}

// Every new line - new metric
// Pattern: {???};{timestamp};{host-name};{service-name};{status};{message}| {metric-name}={value};{warning};{critical}
// TODO: check for matching pattern
func parseBody(body []byte) (*[]transit.MonitoredResource, error) {
	metrics := strings.Split(string(body), "\n")
	var monitoredResources []transit.MonitoredResource
	for _, metric := range metrics {
		re := regexp.MustCompile("[;|=]+")
		arr := re.Split(metric, -1)
		for _, k := range arr {
			fmt.Println(strings.TrimSpace(k))
		}
	}

	return &monitoredResources, nil
}
