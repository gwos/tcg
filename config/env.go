package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var (
	// AllowFlags defines processing the cli arguments
	// true by default
	// false on init of C-shared library libtransit
	AllowFlags = true
	// EnvPrefix defines name prefix for environment variables
	// with struct-path selector and value, for example:
	//    TCG_GWCONNECTIONS_0_PASSWORD=TOK_EN
	EnvPrefix = "TCG_"
	// ConfigEnv defines environment variable for config file path, overrides the ConfigName
	ConfigEnv = "TCG_CONFIG"
	// ConfigName defines default filename for look in work directory if ConfigEnv is empty
	ConfigName = "tcg_config.yaml"
	// SecKeyEnv defines environment variable for
	SecKeyEnv = "TCG_SECKEY"
)

func applyFlags() {
	if !AllowFlags {
		return
	}
	/* as applyFlags (via GetConfig) used to be called in tests init
	and std flag doesn't support it, using github.com/spf13/pflag instead */
	flags := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	flags.StringVar(&EnvPrefix, "env-prefix", "TCG_",
		`prefix for environment variables, "TCG_" by default`)
	flags.StringVar(&ConfigEnv, "config-env", "TCG_CONFIG",
		`environment variable for config file path, "TCG_CONFIG" by default`)
	flags.StringVar(&SecKeyEnv, "seckey-env", "TCG_SECKEY",
		`environment variable for secret to crypt passwords in config file, "TCG_SECKEY" by default`)
	_ = flags.Parse(os.Args[1:])

	for _, s := range []*string{&ConfigEnv, &SecKeyEnv} {
		*s = strings.TrimPrefix(*s, "TCG_")
		*s = strings.TrimPrefix(*s, EnvPrefix)
		*s = EnvPrefix + *s
	}
}

func applyEnv(yamldoc []byte) []byte {
	node := new(yaml.Node)
	if err := yaml.Unmarshal(yamldoc, node); err != nil {
		log.Err(err).
			Str("yamldoc", string(yamldoc)).
			Msg("could not parse yaml")
		return yamldoc
	}

	scalars := map[string]*yaml.Node{}
	nodescan(node, scalars, []string{})

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, EnvPrefix) ||
			strings.HasPrefix(env, ConfigEnv) {
			continue
		}

		pair := strings.SplitN(strings.TrimPrefix(env, EnvPrefix), "=", 2)
		if len(pair) != 2 {
			log.Warn().
				Str("env", env).
				Msg("could not parse env")
			continue
		}

		var key, val = pair[0], pair[1]
		n := scalars[key]
		if n == nil {
			// log.Debug().
			// 	Interface("scalars", scalars). // TODO: remove
			// 	Str("env", env).
			// 	Msg("could not apply env: not found node")
			continue
		}
		if n.ShortTag() == "!!null" {
			n.SetString("")
		}
		n.Value = val
	}

	bb, err := yaml.Marshal(node)
	if err != nil {
		log.Warn().Err(err).
			Msg("could not encode updated yaml")
		return yamldoc
	}
	return bb
}

func nodescan(node *yaml.Node, scalars map[string]*yaml.Node, path []string) {
	var childName string
	for i, n := range node.Content {
		if node.Kind == yaml.SequenceNode {
			childName = fmt.Sprintf("%d", i)

			switch n.Kind {
			case yaml.ScalarNode:
				scalars[strings.ToUpper(strings.Trim(strings.Join(append(path, childName), "_"), "_"))] = n
			case yaml.MappingNode, yaml.SequenceNode:
				nodescan(n, scalars, append(path, childName))
			}
		} else {
			switch n.Kind {
			case yaml.ScalarNode:
				if i%2 == 0 {
					childName = n.Value
				} else {
					scalars[strings.ToUpper(strings.Trim(strings.Join(append(path, childName), "_"), "_"))] = n
				}
			case yaml.MappingNode, yaml.SequenceNode:
				nodescan(n, scalars, append(path, childName))
			}
		}
	}
}
