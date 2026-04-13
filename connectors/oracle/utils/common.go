package utils

import (
	"strings"
)

func getHostName(dimensions map[string]string) (string, bool) {
	for _, key := range []string{
		"displayName",
		"resourceDisplayName",
		"resourceName",
		"hostName",
		"host",
		"nodeName",
		"instanceId",
		"resourceID",
		"resourceId",
	} {
		if value, ok := dimensions[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}
