package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	vtypes "github.com/vultisig/verifier/types"

	"github.com/vultisig/pluginagent/storage/interfaces"
	"github.com/vultisig/pluginagent/storage/postgres/queries"
	"github.com/vultisig/pluginagent/types"
)

var _ interfaces.DatabaseStorage = (*Storage)(nil)

type MigrationOptions struct {
	RunSystemMigrations bool
	RunPluginMigrations bool
}

type Storage struct {
	pool    *pgxpool.Pool
	queries *queries.Queries
}

func NewPostgresStorage(dsn string) (interfaces.DatabaseStorage, error) {
	return NewPostgresStorageWithOptions(dsn, &MigrationOptions{
		RunSystemMigrations: true,
		RunPluginMigrations: true,
	})
}

func NewPostgresStorageWithOptions(dsn string, opts *MigrationOptions) (interfaces.DatabaseStorage, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	backend := &Storage{
		pool:    pool,
		queries: queries.New(pool),
	}

	if opts == nil {
		opts = &MigrationOptions{
			RunSystemMigrations: true,
			RunPluginMigrations: true,
		}
	}

	if err := backend.migrate(opts); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return backend, nil
}

func (s *Storage) Close() error {
	s.pool.Close()
	return nil
}

func (s *Storage) migrate(opts *MigrationOptions) error {
	logrus.Info("Starting database migration...")

	if opts.RunSystemMigrations {
		systemMgr := NewSystemMigrationManager(s.pool)
		if err := systemMgr.Migrate(); err != nil {
			return fmt.Errorf("failed to run system migrations: %w", err)
		}
	}

	if opts.RunPluginMigrations {
		pluginMgr := NewPluginMigrationManager(s.pool)
		if err := pluginMgr.Migrate(); err != nil {
			return fmt.Errorf("failed to run plugin migrations: %w", err)
		}
	}

	logrus.Info("Database migration completed successfully")
	return nil
}

func (s *Storage) GetPluginPolicy(ctx context.Context, id uuid.UUID) (*vtypes.PluginPolicy, error) {
	row, err := s.queries.GetPluginPolicy(ctx, uuidToPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("policy not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	return toVTypesPluginPolicy(row)
}

func (s *Storage) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID vtypes.PluginID, onlyActive bool) ([]vtypes.PluginPolicy, error) {
	params := queries.GetAllPluginPoliciesParams{
		PublicKey: publicKey,
		PluginID:  string(pluginID),
		Column3:   onlyActive,
	}

	rows, err := s.queries.GetAllPluginPolicies(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	policies := make([]vtypes.PluginPolicy, 0, len(rows))
	for _, row := range rows {
		policy, err := toVTypesPluginPolicyFromGetAll(row)
		if err != nil {
			return nil, err
		}
		policies = append(policies, *policy)
	}

	return policies, nil
}

func (s *Storage) InsertPluginPolicy(ctx context.Context, policy vtypes.PluginPolicy) (*vtypes.PluginPolicy, error) {
	params := queries.InsertPluginPolicyParams{
		ID:            uuidToPgUUID(policy.ID),
		PublicKey:     policy.PublicKey,
		PluginID:      string(policy.PluginID),
		PluginVersion: policy.PluginVersion,
		PolicyVersion: fmt.Sprintf("%d", policy.PolicyVersion),
		Signature:     policy.Signature,
		Active:        policy.Active,
		Recipe:        policy.Recipe,
	}

	row, err := s.queries.InsertPluginPolicy(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	return toVTypesPluginPolicyFromInsert(row)
}

func (s *Storage) UpdatePluginPolicy(ctx context.Context, policy vtypes.PluginPolicy) (*vtypes.PluginPolicy, error) {
	params := queries.UpdatePluginPolicyParams{
		ID:            uuidToPgUUID(policy.ID),
		PluginVersion: policy.PluginVersion,
		PolicyVersion: fmt.Sprintf("%d", policy.PolicyVersion),
		Signature:     policy.Signature,
		Active:        policy.Active,
		Recipe:        policy.Recipe,
	}

	row, err := s.queries.UpdatePluginPolicy(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("policy not found with ID: %s", policy.ID)
		}
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	return toVTypesPluginPolicyFromUpdate(row)
}

func (s *Storage) DeletePluginPolicy(ctx context.Context, id uuid.UUID) error {
	err := s.queries.SoftDeletePluginPolicy(ctx, uuidToPgUUID(id))
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}

func (s *Storage) WithTx(ctx context.Context, fn func(interfaces.DatabaseStorage) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			logrus.WithError(err).Error("failed to rollback transaction")
		}
	}()

	txStorage := &Storage{
		pool:    s.pool,
		queries: s.queries.WithTx(tx),
	}

	if err := fn(txStorage); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Storage) InsertEvent(ctx context.Context, event *types.SystemEvent) (int64, error) {
	jsonData, err := json.Marshal(event.EventData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal event data: %w", err)
	}

	var policyID pgtype.UUID
	if event.PolicyID != nil {
		policyID = uuidToPgUUID(*event.PolicyID)
	}

	params := queries.InsertEventParams{
		PublicKey: pgtype.Text{String: *event.PublicKey, Valid: event.PublicKey != nil},
		PolicyID:  policyID,
		EventType: event.EventType,
		EventData: jsonData,
	}

	return s.queries.InsertEvent(ctx, params)
}
