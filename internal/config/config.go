package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	LogLevel string `mapstructure:"log_level"`
}

// Load reads configuration from env vars and optional config file.
// Environment variables are prefixed with AIKITS_ (e.g. AIKITS_LOG_LEVEL).
func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("log_level", "info")

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.aikits")

	v.SetEnvPrefix("AIKITS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Config file is optional; ignore file-not-found errors.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
