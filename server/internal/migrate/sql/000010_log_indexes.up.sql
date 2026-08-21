CREATE INDEX IF NOT EXISTS idx_op_logs_created_at ON op_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_login_logs_created_at ON login_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_api_logs_created_at ON api_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_users_kind_status ON users(kind, status);
