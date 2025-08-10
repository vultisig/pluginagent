package policy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/pluginagent/storage/interfaces"
	"github.com/vultisig/verifier/types"
)

var _ Service = (*Policy)(nil)

type Service interface {
	CreatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	DeletePolicy(ctx context.Context, policyID uuid.UUID, signature string) error
	GetPluginPolicies(
		ctx context.Context,
		pluginID types.PluginID,
		publicKey string,
		onlyActive bool,
	) ([]types.PluginPolicy, error)
	GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (*types.PluginPolicy, error)
}

type Policy struct {
	repo   interfaces.DatabaseStorage
	logger *logrus.Logger
}

func NewPolicyService(
	repo interfaces.DatabaseStorage,
	logger *logrus.Logger,
) (*Policy, error) {
	return &Policy{
		repo:   repo,
		logger: logger.WithField("pkg", "policy").Logger,
	}, nil
}

func (p *Policy) CreatePolicy(c context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	newPolicy, err := p.repo.InsertPluginPolicy(c, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	return newPolicy, nil
}

func (p *Policy) UpdatePolicy(c context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	updatedPolicy, err := p.repo.UpdatePluginPolicy(c, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	return updatedPolicy, nil
}

func (p *Policy) DeletePolicy(c context.Context, policyID uuid.UUID, signature string) error {
	err := p.repo.DeletePluginPolicy(c, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}

func (p *Policy) GetPluginPolicies(
	ctx context.Context,
	pluginID types.PluginID,
	publicKey string,
	onlyActive bool,
) ([]types.PluginPolicy, error) {
	return p.repo.GetAllPluginPolicies(ctx, publicKey, pluginID, onlyActive)
}

func (p *Policy) GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (*types.PluginPolicy, error) {
	return p.repo.GetPluginPolicy(ctx, policyID)
}
