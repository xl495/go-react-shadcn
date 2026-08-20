CREATE TABLE IF NOT EXISTS departments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    parent_id INTEGER,
    sort INTEGER NOT NULL DEFAULT 0,
    leader TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME,
    updated_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id);

ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN failed_login_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN locked_until DATETIME;
ALTER TABLE users ADD COLUMN department_id INTEGER;

ALTER TABLE roles ADD COLUMN data_scope TEXT NOT NULL DEFAULT 'all';

ALTER TABLE permissions ADD COLUMN parent_id INTEGER;
ALTER TABLE permissions ADD COLUMN sort INTEGER NOT NULL DEFAULT 0;
ALTER TABLE permissions ADD COLUMN icon TEXT;
ALTER TABLE permissions ADD COLUMN route_path TEXT;
ALTER TABLE permissions ADD COLUMN component TEXT;
ALTER TABLE permissions ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS login_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    ip TEXT,
    user_agent TEXT,
    location TEXT,
    status TEXT NOT NULL,
    fail_reason TEXT,
    created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_login_logs_username ON login_logs(username);
CREATE INDEX IF NOT EXISTS idx_login_logs_created_at ON login_logs(created_at);

CREATE TABLE IF NOT EXISTS api_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id TEXT NOT NULL,
    username TEXT,
    method TEXT,
    path TEXT,
    status INTEGER,
    latency_ms INTEGER,
    request_body TEXT,
    response_body TEXT,
    error_stack TEXT,
    created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_api_logs_trace_id ON api_logs(trace_id);
CREATE INDEX IF NOT EXISTS idx_api_logs_created_at ON api_logs(created_at);

ALTER TABLE op_logs ADD COLUMN trace_id TEXT;
ALTER TABLE op_logs ADD COLUMN old_value TEXT;
ALTER TABLE op_logs ADD COLUMN new_value TEXT;
ALTER TABLE op_logs ADD COLUMN description TEXT;

CREATE INDEX IF NOT EXISTS idx_op_logs_trace_id ON op_logs(trace_id);
