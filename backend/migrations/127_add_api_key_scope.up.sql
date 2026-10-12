-- Scope each machine API key to exactly one integration surface.
--
-- Until now any key authenticated every /api/v1/integration/* route: a chatbot
-- key implicitly held chatbot-only permissions it never needed. `scope` closes
-- that gap by binding each key to a single purpose, enforced by
-- middleware.APIKeyAuthMiddleware.RequireAPIKeyScope.
--
-- The column is NOT NULL DEFAULT 'chatbot' so every pre-existing key keeps
-- working unchanged after deploy; the explicit UPDATE additionally repairs any
-- row written as NULL or '' before the constraint took effect.
ALTER TABLE api_keys
    ADD COLUMN scope VARCHAR(32) NOT NULL DEFAULT 'chatbot' COMMENT 'chatbot | timesheet_read' AFTER name,
    ADD KEY idx_api_keys_scope (scope);

UPDATE api_keys SET scope = 'chatbot' WHERE scope IS NULL OR scope = '';