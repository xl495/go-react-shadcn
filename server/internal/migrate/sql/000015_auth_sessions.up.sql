CREATE TABLE IF NOT EXISTS auth_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  user_kind TEXT NOT NULL DEFAULT 'admin',
  jti TEXT NOT NULL,
  ip TEXT,
  user_agent TEXT,
  expires_at DATETIME NOT NULL,
  revoked_at DATETIME,
  created_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_sessions_jti ON auth_sessions(jti);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user ON auth_sessions(user_kind, user_id);
