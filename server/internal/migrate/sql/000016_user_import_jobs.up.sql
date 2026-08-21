CREATE TABLE IF NOT EXISTS user_import_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL,
  kind TEXT NOT NULL,
  file_name TEXT,
  status TEXT NOT NULL,
  total INTEGER NOT NULL DEFAULT 0,
  created_count INTEGER NOT NULL DEFAULT 0,
  failed_count INTEGER NOT NULL DEFAULT 0,
  errors TEXT,
  created_at DATETIME,
  updated_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_user_import_jobs_actor ON user_import_jobs(actor_id);
CREATE INDEX IF NOT EXISTS idx_user_import_jobs_status ON user_import_jobs(status);
