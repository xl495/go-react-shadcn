ALTER TABLE users ADD COLUMN kind TEXT NOT NULL DEFAULT 'admin';
CREATE INDEX IF NOT EXISTS idx_users_kind ON users(kind);
