package config

import (
	"os"
	"path"
	"reflect"
	"testing"

	. "github.com/gwos/tng/config"
)

var cfg = Config{
	AgentConfig:      AgentConfig{":8081", "", "", "src/main/resources/datastore", "FILE", true, true, true},
	GroundworkConfig: GroundworkConfig{"localhost:80", "RESTAPIACCESS", "", "gw8"},
}

func TestGetConfig(t *testing.T) {
	os.Setenv(ConfigEnv, path.Join("..", ConfigName))
	os.Setenv("TNG_AGENTCONFIG_NATSSTORETYPE", "MEMORY")
	expected := cfg
	expected.AgentConfig.NATSStoreType = "MEMORY"

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "envconfig",
			want: &expected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
