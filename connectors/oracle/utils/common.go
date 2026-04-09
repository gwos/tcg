package utils

import (
	"strings"
)

func getHostName(dimensions map[string]string) (string, bool) {
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
			name := strings.TrimSpace(value)
			if strings.HasPrefix(strings.ToLower(name), "ocid1.") {
				continue
			}
			return name, true
		}
	}
	return "", false
}
