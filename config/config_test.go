package config

import (
	"os"
	"path"
	"reflect"
	"testing"
)

func TestGetConfig(t *testing.T) {
	expected := Config{
		AgentConfig: AgentConfig{":8081", "", "", 15, "src/main/resources/datastore", "MEMORY", ":4222", true, true, true, 3},
		GWConfig:    GWConfig{"localhost:80", "RESTAPIACCESS", "SECRET", "gw8"},
	}

	os.Setenv(ConfigEnv, path.Join("..", ConfigName))
	os.Setenv("TNG_AGENTCONFIG_NATSSTORETYPE", "MEMORY")
	os.Setenv("TNG_GWCONFIG_PASSWORD", "SECRET")

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
