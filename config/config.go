package config

import (
	"github.com/spf13/viper"
	"github.com/vultisig/verifier/vault_config"
)

type Config struct {
	Redis        RedisConfig               `mapstructure:"redis" json:"redis"`
	VaultService vault_config.Config       `mapstructure:"vault_service" json:"vault_service,omitempty"`
	BlockStorage vault_config.BlockStorage `mapstructure:"block_storage" json:"block_storage,omitempty"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host" json:"host"`
	Port     string `mapstructure:"port" json:"port"`
	User     string `mapstructure:"user" json:"user"`
	Password string `mapstructure:"password" json:"password"`
	DB       int    `mapstructure:"db" json:"db"`
}

func LoadWorkerConfig() (*Config, error) {
	cfg := &Config{}

	viper.SetConfigName("worker")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
