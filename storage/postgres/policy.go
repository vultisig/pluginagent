package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	vtypes "github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetPluginPolicy(ctx context.Context, id uuid.UUID) (*vtypes.PluginPolicy, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	var policy vtypes.PluginPolicy
	query := `
        SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active,  recipe
        FROM plugin_policies 
        WHERE id = $1`

	err := p.pool.QueryRow(ctx, query, id).Scan(
		&policy.ID,
		&policy.PublicKey,
		&policy.PluginID,
		&policy.PluginVersion,
		&policy.PolicyVersion,
		&policy.Signature,
		&policy.Active,
		&policy.Recipe,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	return &policy, nil
}

func (p *PostgresBackend) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID vtypes.PluginID, onlyActive bool) ([]vtypes.PluginPolicy, error) {

	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
  	SELECT id, public_key,  plugin_id, plugin_version, policy_version, signature, active, recipe
		FROM plugin_policies
		WHERE public_key = $1
		AND plugin_id = $2`

	if onlyActive {
		query += ` AND active = true`
	}

	rows, err := p.pool.Query(ctx, query, publicKey, pluginID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []vtypes.PluginPolicy
	for rows.Next() {
		var policy vtypes.PluginPolicy
		err := rows.Scan(
			&policy.ID,
			&policy.PublicKey,
			&policy.PluginID,
			&policy.PluginVersion,
			&policy.PolicyVersion,
			&policy.Signature,
			&policy.Active,
			&policy.Recipe,
		)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

func (p *PostgresBackend) InsertPluginPolicy(ctx context.Context, policy vtypes.PluginPolicy) (*vtypes.PluginPolicy, error) {
	query := `
  	INSERT INTO plugin_policies (
      id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, public_key,  plugin_id, plugin_version, policy_version, signature, active, recipe
	`

	var insertedPolicy vtypes.PluginPolicy
	err := p.pool.QueryRow(ctx, query,
		policy.ID,
		policy.PublicKey,
		policy.PluginID,
		policy.PluginVersion,
		policy.PolicyVersion,
		policy.Signature,
		policy.Active,
		policy.Recipe,
	).Scan(
		&insertedPolicy.ID,
		&insertedPolicy.PublicKey,
		&insertedPolicy.PluginID,
		&insertedPolicy.PluginVersion,
		&insertedPolicy.PolicyVersion,
		&insertedPolicy.Signature,
		&insertedPolicy.Active,
		&insertedPolicy.Recipe,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	return &insertedPolicy, nil
}

func (p *PostgresBackend) UpdatePluginPolicy(ctx context.Context, policy vtypes.PluginPolicy) (*vtypes.PluginPolicy, error) {
	query := `
		UPDATE plugin_policies 
		SET plugin_version = $2,
		    policy_version = $3,
			signature = $4,
			active = $5,
			recipe = $6
		WHERE id = $1
		RETURNING id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
	`

	var updatedPolicy vtypes.PluginPolicy
	err := p.pool.QueryRow(ctx, query,
		policy.ID,
		policy.PluginVersion,
		policy.PolicyVersion,
		policy.Signature,
		policy.Active,
		policy.Recipe,
	).Scan(
		&updatedPolicy.ID,
		&updatedPolicy.PublicKey,
		&updatedPolicy.PluginID,
		&updatedPolicy.PluginVersion,
		&updatedPolicy.PolicyVersion,
		&updatedPolicy.Signature,
		&updatedPolicy.Active,
		&updatedPolicy.Recipe,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("policy not found with ID: %s", policy.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	return &updatedPolicy, nil
}

func (p *PostgresBackend) DeletePluginPolicy(ctx context.Context, id uuid.UUID) error {
	_, err := p.pool.Exec(ctx, `
	DELETE FROM time_triggers
	WHERE policy_id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete time triggers: %w", err)
	}
	_, err = p.pool.Exec(ctx, `
	UPDATE plugin_policies
	SET deleted = true
	WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}
