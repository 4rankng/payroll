-- Scope-toggle support for MCP machine keys (mcp_*): records when an admin
-- last flipped the scope (read_only <-> full) so the admin list can show the
-- real change time. NULL = the key still carries its creation-time scope.
ALTER TABLE mcp_api_keys
    ADD COLUMN scope_updated_at DATETIME(3) NULL COMMENT 'last admin scope toggle (read_only <-> full)' AFTER revoked_at;
