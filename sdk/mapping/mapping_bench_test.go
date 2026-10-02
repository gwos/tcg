package mapping

import (
	"encoding/json"
	"testing"
)

func benchMappings(b *testing.B, data string) Mappings {
	b.Helper()
	var v struct{ Mappings Mappings }
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		b.Fatal(err)
	}
	if err := v.Mappings.Compile(); err != nil {
		b.Fatal(err)
	}
	return v.Mappings
}

func BenchmarkMappingsApplyOR(b *testing.B) {
	tagMaps := testData()
	for _, bc := range []struct{ name, mappings string }{
		{"single", `{"mappings":[{"tag":"pod_name","matcher":"(.*)","template":"pod-$1"},{"tag":"node_name","matcher":"(.*)","template":"node-$1"},{"tag":"","template":"default"}]}`},
		{"multi", `{"mappings":[{"tag":"cluster,namespace","matcher":"(.*),(.*)","template":"$1-$2"},{"tag":"","template":"default"}]}`},
	} {
		m := benchMappings(b, bc.mappings)
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				for _, tags := range tagMaps {
					_, _ = m.ApplyOR(tags)
				}
			}
		})
	}
}

func BenchmarkMappingsApply(b *testing.B) {
	tagMaps := testData()
	m := benchMappings(b, `{"mappings":[{"tag":"cluster","matcher":"(.*)","template":"$1"},{"tag":"","template":"/"},{"tag":"source","matcher":"([^|]*)$","template":"$1"}]}`)
	b.ReportAllocs()
	for b.Loop() {
		for _, tags := range tagMaps {
			_, _ = m.Apply(tags)
		}
	}
}

func BenchmarkMappingsMatchString(b *testing.B) {
	m := benchMappings(b, `{"mappings":[{"tag":"x","matcher":"^kube-","template":""},{"tag":"x","matcher":"proxy","template":""}]}`)
	strs := []string{"kube-proxy", "coredns", "metrics-server", "registry-proxy"}
	b.ReportAllocs()
	for b.Loop() {
		for _, s := range strs {
			_ = m.MatchString(s)
		}
	}
}
