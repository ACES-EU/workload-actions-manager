package config

import (
	"bytes"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// env variable example: REDIS_PORT=1234

type Config struct {
	Server Server `mapstructure:"SERVER"`
	Redis  Redis  `mapstructure:"REDIS"`
}

type Server struct {
	Address string `mapstructure:"ADDRESS"`
}

type Redis struct {
	Host     string `mapstructure:"HOST"`
	Port     string `mapstructure:"PORT"`
	Password string `mapstructure:"PASSWORD"`
}

func defaultConfig() *Config {
	return &Config{}
}

func New() (*Config, error) {
	v := viper.New()

	configBytes, err := yaml.Marshal(defaultConfig())
	if err != nil {
		return nil, err
	}

	// set defaults
	confReader := bytes.NewReader(configBytes)
	v.SetConfigType("yaml")
	if err := v.MergeConfig(confReader); err != nil {
		return nil, err
	}

	// override with env variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	config := &Config{}
	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
