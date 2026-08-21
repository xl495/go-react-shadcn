CREATE TABLE IF NOT EXISTS notifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_kind TEXT NOT NULL,
  user_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  ref_type TEXT NOT NULL DEFAULT '',
  ref_id INTEGER NOT NULL DEFAULT 0,
  read_at DATETIME,
  created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_notifications_inbox ON notifications (user_kind, user_id, read_at, created_at);
