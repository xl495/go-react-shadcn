ALTER TABLE login_logs ADD COLUMN user_kind TEXT NOT NULL DEFAULT 'admin';
CREATE INDEX IF NOT EXISTS idx_login_logs_kind_username ON login_logs (user_kind, username);
