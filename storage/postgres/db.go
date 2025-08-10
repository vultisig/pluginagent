package postgres

import (
	"github.com/vultisig/pluginagent/storage/interfaces"
)

type MigrationOptions struct {
	RunSystemMigrations bool
	RunPluginMigrations bool
}

func NewPostgresBackend(dsn string, opts *MigrationOptions) (interfaces.DatabaseStorage, error) {
	return NewPostgresStorage(dsn)
}