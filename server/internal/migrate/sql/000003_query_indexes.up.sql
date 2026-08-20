CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_op_logs_action ON op_logs(action);
CREATE INDEX IF NOT EXISTS idx_login_logs_status ON login_logs(status);
