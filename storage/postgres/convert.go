package postgres

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	vtypes "github.com/vultisig/verifier/types"

	"github.com/vultisig/pluginagent/storage/postgres/queries"
	"github.com/vultisig/pluginagent/types"
)

func toVTypesPluginPolicy(row queries.GetPluginPolicyRow) (*vtypes.PluginPolicy, error) {
	id, err := uuidFromPgUUID(row.ID)
	if err != nil {
		return nil, err
	}

	policyVersion, err := strconv.Atoi(row.PolicyVersion)
	if err != nil {
		return nil, err
	}

	return &vtypes.PluginPolicy{
		ID:            id,
		PublicKey:     row.PublicKey,
		PluginID:      vtypes.PluginID(row.PluginID),
		PluginVersion: row.PluginVersion,
		PolicyVersion: policyVersion,
		Signature:     row.Signature,
		Active:        row.Active,
		Recipe:        row.Recipe,
	}, nil
}

func toVTypesPluginPolicyFromInsert(row queries.InsertPluginPolicyRow) (*vtypes.PluginPolicy, error) {
	id, err := uuidFromPgUUID(row.ID)
	if err != nil {
		return nil, err
	}

	policyVersion, err := strconv.Atoi(row.PolicyVersion)
	if err != nil {
		return nil, err
	}

	return &vtypes.PluginPolicy{
		ID:            id,
		PublicKey:     row.PublicKey,
		PluginID:      vtypes.PluginID(row.PluginID),
		PluginVersion: row.PluginVersion,
		PolicyVersion: policyVersion,
		Signature:     row.Signature,
		Active:        row.Active,
		Recipe:        row.Recipe,
	}, nil
}

func toVTypesPluginPolicyFromUpdate(row queries.UpdatePluginPolicyRow) (*vtypes.PluginPolicy, error) {
	id, err := uuidFromPgUUID(row.ID)
	if err != nil {
		return nil, err
	}

	policyVersion, err := strconv.Atoi(row.PolicyVersion)
	if err != nil {
		return nil, err
	}

	return &vtypes.PluginPolicy{
		ID:            id,
		PublicKey:     row.PublicKey,
		PluginID:      vtypes.PluginID(row.PluginID),
		PluginVersion: row.PluginVersion,
		PolicyVersion: policyVersion,
		Signature:     row.Signature,
		Active:        row.Active,
		Recipe:        row.Recipe,
	}, nil
}

func toVTypesPluginPolicyFromGetAll(row queries.GetAllPluginPoliciesRow) (*vtypes.PluginPolicy, error) {
	id, err := uuidFromPgUUID(row.ID)
	if err != nil {
		return nil, err
	}

	policyVersion, err := strconv.Atoi(row.PolicyVersion)
	if err != nil {
		return nil, err
	}

	return &vtypes.PluginPolicy{
		ID:            id,
		PublicKey:     row.PublicKey,
		PluginID:      vtypes.PluginID(row.PluginID),
		PluginVersion: row.PluginVersion,
		PolicyVersion: policyVersion,
		Signature:     row.Signature,
		Active:        row.Active,
		Recipe:        row.Recipe,
	}, nil
}

func toTypesSystemEvent(row queries.SystemEvent) (*types.SystemEvent, error) {
	var policyID *uuid.UUID
	if row.PolicyID.Valid {
		policyIDParsed, err := uuidFromPgUUID(row.PolicyID)
		if err != nil {
			return nil, err
		}
		policyID = &policyIDParsed
	}

	var publicKey *string
	if row.PublicKey.Valid {
		publicKey = &row.PublicKey.String
	}

	return &types.SystemEvent{
		ID:        row.ID,
		PublicKey: publicKey,
		PolicyID:  policyID,
		EventType: types.SystemEventType(row.EventType),
		EventData: row.EventData,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

func uuidFromPgUUID(pguuid pgtype.UUID) (uuid.UUID, error) {
	if !pguuid.Valid {
		return uuid.Nil, nil
	}
	return pguuid.Bytes, nil
}
