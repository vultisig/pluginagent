-- name: GetPluginPolicy :one
SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
FROM plugin_policies 
WHERE id = $1;

-- name: GetAllPluginPolicies :many
SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
FROM plugin_policies
WHERE public_key = $1
  AND plugin_id = $2
  AND ($3::boolean = false OR active = true);

-- name: InsertPluginPolicy :one
INSERT INTO plugin_policies (
    id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe;

-- name: UpdatePluginPolicy :one
UPDATE plugin_policies 
SET plugin_version = $2,
    policy_version = $3,
    signature = $4,
    active = $5,
    recipe = $6
WHERE id = $1
RETURNING id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe;

-- name: SoftDeletePluginPolicy :exec
UPDATE plugin_policies
SET deleted = true
WHERE id = $1;