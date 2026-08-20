ALTER TABLE users ADD COLUMN google_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
