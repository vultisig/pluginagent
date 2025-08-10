CREATE TABLE IF NOT EXISTS plugin_policies (
    id UUID PRIMARY KEY,
    public_key TEXT NOT NULL,
    plugin_id TEXT NOT NULL,
    plugin_version TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    signature TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    recipe TEXT NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT false
);