package utils

import (
	"strings"
)

func getHostName(dimensions map[string]string, fallback string) string {
	for _, key := range []string{
		"resourceDisplayName",
		"resourceName",
		"resourceId",
		"resourceID",
		"instanceId",
		"hostName",
		"host",
		"nodeName",
		"displayName",
	} {
		if value, ok := dimensions[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "unnamed-oracle-resource"
}
