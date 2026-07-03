ALTER TABLE users DROP COLUMN IF EXISTS token_version;
DROP TABLE IF EXISTS oauth_connections;
DROP TABLE IF EXISTS refresh_tokens;