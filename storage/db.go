package storage

import (
	"context"

	"github.com/google/uuid"
	vtypes "github.com/vultisig/verifier/types"
)

type DatabaseStorage interface {
	Close() error

	GetPluginPolicy(ctx context.Context, id uuid.UUID) (*vtypes.PluginPolicy, error)
	GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID vtypes.PluginID, onlyActive bool) ([]vtypes.PluginPolicy, error)
	DeletePluginPolicy(ctx context.Context, id uuid.UUID) error
	InsertPluginPolicy(ctx context.Context, policy vtypes.PluginPolicy) (*vtypes.PluginPolicy, error)
	UpdatePluginPolicy(ctx context.Context, policy vtypes.PluginPolicy) (*vtypes.PluginPolicy, error)
}
