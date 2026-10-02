package mapping

import (
	"errors"
	"testing"
)

const (
	matchAll   = "(.*)"
	tagCluster = "cluster"
)

func TestMappingsLookup(t *testing.T) {
	tags := map[string]string{"host": "host01.example.com", "empty": ""}
	cases := []struct {
		name     string
		mappings Mappings
		val      string
		ok       bool
		err      error
	}{
		{"unconfigured", nil, "", false, nil},
		{"matched", Mappings{*NewMapping("host", `^([^.]+)`, "$1")}, "host01", true, nil},
		{"missed", Mappings{*NewMapping("missing", matchAll, "$1")}, "", false, ErrMappingMissedTag},
		{"mismatched", Mappings{*NewMapping("host", `^\d+$`, "$0")}, "", false, ErrMappingMismatchedTag},
		{"empty result", Mappings{*NewMapping("empty", matchAll, "$1")}, "", false, ErrMappingEmptyResult},
		{"empty constant", Mappings{*NewMapping("", "", "")}, "", false, ErrMappingEmptyResult},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, ok, err := tc.mappings.Lookup(tags)
			if val != tc.val || ok != tc.ok || !errors.Is(err, tc.err) {
				t.Errorf("Lookup() = %q, %v, %v; want %q, %v, %v", val, ok, err, tc.val, tc.ok, tc.err)
			}
		})
	}
}

func TestMappingsCompileValid(t *testing.T) {
	mappings := Mappings{
		{Tag: "a", Matcher: matchAll, Template: "$1"},
		{Tag: "b", Matcher: "(", Template: "$1"},
		{Tag: "c", Matcher: "^c$", Template: "$0"},
	}
	valid, err := mappings.CompileValid()
	if !errors.Is(err, ErrMappingCompile) {
		t.Errorf("CompileValid() error = %v; want %v", err, ErrMappingCompile)
	}
	if len(valid) != 2 || valid[0].Tag != "a" || valid[1].Tag != "c" {
		t.Fatalf("CompileValid() = %+v; want mappings a and c", valid)
	}
	if v, err := valid.ApplyOR(map[string]string{"c": "c"}); v != "c" || err != nil {
		t.Errorf("ApplyOR() = %q, %v; want %q, nil", v, err, "c")
	}

	valid, err = Mappings{{Tag: "a", Matcher: matchAll}}.CompileValid()
	if err != nil || len(valid) != 1 {
		t.Errorf("CompileValid() = %+v, %v; want 1 mapping, nil", valid, err)
	}
}

func TestMappingsNotCompiled(t *testing.T) {
	mappings := Mappings{{Tag: "a", Matcher: matchAll, Template: "$1"}}
	tags := map[string]string{"a": "a"}
	if _, err := mappings.ApplyOR(tags); !errors.Is(err, ErrMappingCompile) {
		t.Errorf("ApplyOR() error = %v; want %v", err, ErrMappingCompile)
	}
	if _, err := mappings.Apply(tags); !errors.Is(err, ErrMappingCompile) {
		t.Errorf("Apply() error = %v; want %v", err, ErrMappingCompile)
	}
	if mappings.MatchString("a") {
		t.Error("MatchString() = true; want false")
	}
}

func TestMappingsApplyORMultiTag(t *testing.T) {
	mappings := Mappings{*NewMapping("cluster, namespace", "^(.+),(.+)$", "$1-$2")}
	if v, err := mappings.ApplyOR(map[string]string{tagCluster: "prod", "namespace": "web"}); v != "prod-web" || err != nil {
		t.Errorf("ApplyOR() = %q, %v; want %q, nil", v, err, "prod-web")
	}
	if _, err := mappings.ApplyOR(map[string]string{tagCluster: "prod"}); !errors.Is(err, ErrMappingMissedTag) {
		t.Errorf("ApplyOR() error = %v; want %v", err, ErrMappingMissedTag)
	}
}

func TestMappingsMatches(t *testing.T) {
	tags := map[string]string{"message": "DB client connected"}
	cases := []struct {
		name     string
		mappings Mappings
		want     bool
	}{
		{"unconfigured", nil, false},
		{"matched", Mappings{*NewMapping("message", "client connected", "ignore")}, true},
		{"mismatched", Mappings{*NewMapping("message", "shutting down", "ignore")}, false},
		{"missed", Mappings{*NewMapping("missing", matchAll, "ignore")}, false},
		{"empty result", Mappings{*NewMapping("message", "client connected", "")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.mappings.Matches(tags); got != tc.want {
				t.Errorf("Matches() = %v; want %v", got, tc.want)
			}
		})
	}
}
