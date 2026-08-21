ALTER TABLE admin_user ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE admin_user ADD COLUMN totp_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_user ADD COLUMN totp_verified_at DATETIME;
ALTER TABLE web_user ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE web_user ADD COLUMN totp_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE web_user ADD COLUMN totp_verified_at DATETIME;

CREATE TABLE IF NOT EXISTS totp_recovery_codes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  user_kind TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  used_at DATETIME,
  created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_totp_recovery_user ON totp_recovery_codes (user_kind, user_id);
