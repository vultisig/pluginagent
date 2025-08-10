package storage

import (
	"fmt"

	"github.com/vultisig/pluginagent/storage/interfaces"
	"github.com/vultisig/pluginagent/storage/postgres"
)

type StorageType string

const (
	StorageTypePostgreSQL StorageType = "postgresql"
	StorageTypeSQLite     StorageType = "sqlite"
)

type StorageConfig struct {
	Type StorageType
	DSN  string
}

// NewDatabaseStorage creates a new database storage instance based on the config.
func NewDatabaseStorage(config StorageConfig) (interfaces.DatabaseStorage, error) {
	switch config.Type {
	case StorageTypePostgreSQL:
		return postgres.NewPostgresStorage(config.DSN)
	case StorageTypeSQLite:
		return nil, fmt.Errorf("sqlite storage not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}

