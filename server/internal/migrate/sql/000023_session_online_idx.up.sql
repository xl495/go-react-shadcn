CREATE INDEX IF NOT EXISTS idx_auth_sessions_online ON auth_sessions (expires_at) WHERE revoked_at IS NULL;
