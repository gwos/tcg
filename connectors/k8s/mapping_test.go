package k8s

import (
	"testing"

	"github.com/gwos/tcg/sdk/mapping"
	"github.com/stretchr/testify/assert"
)

const (
	matchAll = "(.*)"
	tagApp   = "app"
	tagNode  = "node_name"
	podName  = "pod01"
	podGroup = "pods-kube-system"
)

func TestGWMappingMapNames(t *testing.T) {
	labels := map[string]string{"namespace": "kube-system", tagNode: "minikube", tagApp: ""}
	cases := []struct {
		name              string
		mapping           GWMapping
		wantName, wantGrp string
		wantOK            bool
	}{
		{"unconfigured", GWMapping{}, podName, podGroup, true},
		{"mapped", GWMapping{
			HostName:  mapping.Mappings{{Tag: tagNode, Matcher: matchAll, Template: "node-$1"}},
			HostGroup: mapping.Mappings{{Tag: "namespace", Matcher: matchAll, Template: "ns-$1"}},
		}, "node-minikube", "ns-kube-system", true},
		{"hostname missed", GWMapping{
			HostName: mapping.Mappings{{Tag: "pod_name", Matcher: matchAll, Template: "$1"}},
		}, "", "", false},
		{"hostname empty", GWMapping{
			HostName: mapping.Mappings{{Tag: tagApp, Matcher: matchAll, Template: "$1"}},
		}, "", "", false},
		{"ignored", GWMapping{
			Ignore: mapping.Mappings{{Tag: tagNode, Matcher: "^mini", Template: "ignore"}},
		}, "", "", false},
		{"not ignored", GWMapping{
			Ignore: mapping.Mappings{{Tag: tagNode, Matcher: "^prod", Template: "ignore"}},
		}, podName, podGroup, true},
		{"hostgroup falls back", GWMapping{
			HostGroup: mapping.Mappings{{Tag: tagApp, Matcher: matchAll, Template: "$1"}},
		}, podName, podGroup, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mapping.Prepare()
			name, group, ok := tc.mapping.mapNames(labels, podName, "pods-", "pod")
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantName, name)
			assert.Equal(t, tc.wantGrp, group)
		})
	}
}

func TestGWMappingPrepareDropsInvalid(t *testing.T) {
	m := GWMapping{HostName: mapping.Mappings{
		{Tag: "a", Matcher: "(", Template: "$1"},
		{Tag: "b", Matcher: matchAll, Template: "$1"},
	}}
	m.Prepare()
	assert.Len(t, m.HostName, 1)
	assert.Equal(t, "b", m.HostName[0].Tag)
}
