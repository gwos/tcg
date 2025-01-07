package config

import (
	"errors"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/spf13/pflag"
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

func applyEnv(v ...interface{}) error {
	var ee []error
	for i := range v {
		if err := env.ParseWithOptions(v[i], env.Options{Prefix: EnvPrefix}); err != nil {
			ee = append(ee, err)
		}
	}
	if len(ee) > 0 {
		return errors.Join(ee...)
	}
	return nil
}
