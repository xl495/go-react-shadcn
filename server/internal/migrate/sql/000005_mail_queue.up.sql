CREATE TABLE IF NOT EXISTS mail_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER,
    class TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 5,
    user_id INTEGER,
    to_email TEXT NOT NULL,
    timezone TEXT,
    subject TEXT NOT NULL,
    body TEXT,
    status TEXT NOT NULL DEFAULT 'queued',
    send_after DATETIME NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    dedupe_key TEXT,
    sent_at DATETIME,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_mail_jobs_status_send ON mail_jobs(status, send_after, priority, id);
CREATE INDEX IF NOT EXISTS idx_mail_jobs_campaign_id ON mail_jobs(campaign_id);
CREATE INDEX IF NOT EXISTS idx_mail_jobs_dedupe_key ON mail_jobs(dedupe_key);

CREATE TABLE IF NOT EXISTS mail_campaigns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT,
    audience TEXT NOT NULL DEFAULT 'opted_in',
    status TEXT NOT NULL DEFAULT 'draft',
    scheduled_at DATETIME,
    started_at DATETIME,
    finished_at DATETIME,
    job_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_mail_campaigns_status ON mail_campaigns(status);
