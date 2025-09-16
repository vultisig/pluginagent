package config

import (
	"github.com/spf13/viper"
	"github.com/vultisig/verifier/vault_config"
)

type Config struct {
	Redis        RedisConfig               `mapstructure:"redis" json:"redis"`
	VaultService vault_config.Config       `mapstructure:"vault_service" json:"vault_service,omitempty"`
	BlockStorage vault_config.BlockStorage `mapstructure:"block_storage" json:"block_storage,omitempty"`
	Server       ServerConfig              `mapstructure:"server" json:"server,omitempty"`
	Database     DatabaseConfig            `mapstructure:"database" json:"database,omitempty"`
	Plugin       PluginConfig              `mapstructure:"plugin" json:"plugin,omitempty"`
	Verifier     VerifierConfig            `mapstructure:"verifier" json:"verifier,omitempty"`
}

type VerifierConfig struct {
	URL    string `mapstructure:"url" json:"url,omitempty"`
	Token  string `mapstructure:"token" json:"token,omitempty"`
	Prefix string `mapstructure:"prefix" json:"prefix,omitempty"`
}

type PluginConfig struct {
	PluginID                    string `mapstructure:"plugin_id" json:"plugin_id,omitempty"`
	RecipeSpecificationFilePath string `mapstructure:"recipe_specification_file_path" json:"recipe_specification_file_path,omitempty"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
}

type ServerConfig struct {
	Host             string `mapstructure:"host" json:"host,omitempty"`
	Port             int64  `mapstructure:"port" json:"port,omitempty"`
	EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
	VaultsFilePath   string `mapstructure:"vaults_file_path" json:"vaults_file_path,omitempty"` //This is just for testing locally
}

type RedisConfig struct {
	Host     string `mapstructure:"host" json:"host"`
	Port     string `mapstructure:"port" json:"port"`
	User     string `mapstructure:"user" json:"user"`
	Password string `mapstructure:"password" json:"password"`
	DB       int    `mapstructure:"db" json:"db"`
}

func LoadServerConfig() (*Config, error) {
	cfg := &Config{}

	viper.SetConfigName("agent")
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

func LoadWorkerConfig() (*Config, error) {
	cfg := &Config{}

	viper.SetConfigName("agent")
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
