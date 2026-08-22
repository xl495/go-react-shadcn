CREATE TABLE IF NOT EXISTS roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    description TEXT,
    data_scope TEXT NOT NULL DEFAULT 'all',
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    path TEXT NOT NULL,
    method TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'api',
    description TEXT,
    parent_id INTEGER,
    sort INTEGER NOT NULL DEFAULT 0,
    icon TEXT,
    route_path TEXT,
    component TEXT,
    hidden INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS dict_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    remark TEXT,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS dict_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type_code TEXT NOT NULL,
    label TEXT NOT NULL,
    value TEXT NOT NULL,
    sort INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    remark TEXT,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_dict_items_type_code ON dict_items(type_code);

CREATE TABLE IF NOT EXISTS sys_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    name TEXT NOT NULL,
    "group" TEXT,
    remark TEXT,
    created_at DATETIME,
    updated_at DATETIME
);

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

CREATE TABLE IF NOT EXISTS admin_user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nickname TEXT,
    avatar TEXT,
    email TEXT,
    phone TEXT,
    gender TEXT,
    department TEXT,
    department_id INTEGER,
    title TEXT,
    remark TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    token_version INTEGER NOT NULL DEFAULT 0,
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until DATETIME,
    last_login_at DATETIME,
    last_login_ip TEXT,
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    marketing_opt_in INTEGER NOT NULL DEFAULT 1,
    google_id TEXT NOT NULL DEFAULT '',
    email_verified INTEGER NOT NULL DEFAULT 1,
    totp_enabled INTEGER NOT NULL DEFAULT 0,
    totp_secret TEXT NOT NULL DEFAULT '',
    totp_verified_at DATETIME,
    must_change_password INTEGER NOT NULL DEFAULT 0,
    pending_email TEXT NOT NULL DEFAULT '',
    must_set_password INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_admin_user_department_id ON admin_user(department_id);
CREATE INDEX IF NOT EXISTS idx_admin_user_google_id ON admin_user(google_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_user_email_lower ON admin_user(lower(email)) WHERE email <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_user_google_id_nonempty ON admin_user(google_id) WHERE google_id <> '';

CREATE TABLE IF NOT EXISTS web_user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nickname TEXT,
    avatar TEXT,
    email TEXT,
    phone TEXT,
    gender TEXT,
    department TEXT,
    department_id INTEGER,
    title TEXT,
    remark TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    token_version INTEGER NOT NULL DEFAULT 0,
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until DATETIME,
    last_login_at DATETIME,
    last_login_ip TEXT,
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    marketing_opt_in INTEGER NOT NULL DEFAULT 1,
    google_id TEXT NOT NULL DEFAULT '',
    email_verified INTEGER NOT NULL DEFAULT 1,
    totp_enabled INTEGER NOT NULL DEFAULT 0,
    totp_secret TEXT NOT NULL DEFAULT '',
    totp_verified_at DATETIME,
    must_change_password INTEGER NOT NULL DEFAULT 0,
    pending_email TEXT NOT NULL DEFAULT '',
    must_set_password INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_web_user_department_id ON web_user(department_id);
CREATE INDEX IF NOT EXISTS idx_web_user_google_id ON web_user(google_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_web_user_email_lower ON web_user(lower(email)) WHERE email <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_web_user_google_id_nonempty ON web_user(google_id) WHERE google_id <> '';

CREATE TABLE IF NOT EXISTS web_user_roles (
    user_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_web_user_roles_role_id ON web_user_roles(role_id);

CREATE VIRTUAL TABLE IF NOT EXISTS admin_user_fts USING fts5(
  username, nickname, email, phone, content='admin_user', content_rowid='id', prefix='2 3 4'
);
CREATE VIRTUAL TABLE IF NOT EXISTS web_user_fts USING fts5(
  username, nickname, email, phone, content='web_user', content_rowid='id', prefix='2 3 4'
);

CREATE TRIGGER IF NOT EXISTS admin_user_fts_ai AFTER INSERT ON admin_user BEGIN
  INSERT INTO admin_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;
CREATE TRIGGER IF NOT EXISTS admin_user_fts_ad AFTER DELETE ON admin_user BEGIN
  INSERT INTO admin_user_fts(admin_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
END;
CREATE TRIGGER IF NOT EXISTS admin_user_fts_au AFTER UPDATE ON admin_user BEGIN
  INSERT INTO admin_user_fts(admin_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
  INSERT INTO admin_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;
CREATE TRIGGER IF NOT EXISTS web_user_fts_ai AFTER INSERT ON web_user BEGIN
  INSERT INTO web_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;
CREATE TRIGGER IF NOT EXISTS web_user_fts_ad AFTER DELETE ON web_user BEGIN
  INSERT INTO web_user_fts(web_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
END;
CREATE TRIGGER IF NOT EXISTS web_user_fts_au AFTER UPDATE ON web_user BEGIN
  INSERT INTO web_user_fts(web_user_fts, rowid, username, nickname, email, phone)
  VALUES ('delete', old.id, old.username, old.nickname, old.email, old.phone);
  INSERT INTO web_user_fts(rowid, username, nickname, email, phone)
  VALUES (new.id, new.username, new.nickname, new.email, new.phone);
END;

CREATE TABLE IF NOT EXISTS nav_menu (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER,
    audience TEXT NOT NULL,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    route_path TEXT,
    component TEXT,
    icon TEXT,
    sort INTEGER NOT NULL DEFAULT 0,
    hidden INTEGER NOT NULL DEFAULT 0,
    perm_code TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_nav_audience_code ON nav_menu(audience, code);

CREATE TABLE IF NOT EXISTS op_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id TEXT,
    username TEXT,
    module TEXT,
    action TEXT,
    method TEXT,
    path TEXT,
    status INTEGER,
    ip TEXT,
    latency_ms INTEGER,
    detail TEXT,
    description TEXT,
    old_value TEXT,
    new_value TEXT,
    created_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_op_logs_username ON op_logs(username);
CREATE INDEX IF NOT EXISTS idx_op_logs_module ON op_logs(module);
CREATE INDEX IF NOT EXISTS idx_op_logs_action ON op_logs(action);
CREATE INDEX IF NOT EXISTS idx_op_logs_created_at ON op_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_op_logs_trace_id ON op_logs(trace_id);

CREATE TABLE IF NOT EXISTS login_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    user_kind TEXT NOT NULL DEFAULT 'admin',
    ip TEXT,
    user_agent TEXT,
    location TEXT,
    status TEXT NOT NULL,
    fail_reason TEXT,
    created_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_login_logs_username ON login_logs(username);
CREATE INDEX IF NOT EXISTS idx_login_logs_created_at ON login_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_login_logs_status ON login_logs(status);
CREATE INDEX IF NOT EXISTS idx_login_logs_kind_username ON login_logs(user_kind, username);

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

CREATE TABLE IF NOT EXISTS casbin_rule (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ptype TEXT,
    v0 TEXT,
    v1 TEXT,
    v2 TEXT,
    v3 TEXT,
    v4 TEXT,
    v5 TEXT
);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_ptype ON casbin_rule(ptype);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    user_kind TEXT NOT NULL DEFAULT 'admin',
    purpose TEXT NOT NULL DEFAULT 'reset',
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

CREATE TABLE IF NOT EXISTS mail_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER,
    class TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 5,
    user_id INTEGER,
    user_kind TEXT NOT NULL DEFAULT '',
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
CREATE INDEX IF NOT EXISTS idx_auth_sessions_online ON auth_sessions(expires_at) WHERE revoked_at IS NULL;

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

CREATE TABLE IF NOT EXISTS captcha_challenges (
    id TEXT PRIMARY KEY,
    answer TEXT NOT NULL,
    expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_captcha_challenges_expires ON captcha_challenges(expires_at);

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
CREATE INDEX IF NOT EXISTS idx_notifications_inbox ON notifications(user_kind, user_id, read_at, created_at);

CREATE TABLE IF NOT EXISTS totp_recovery_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    user_kind TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    used_at DATETIME,
    created_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_totp_recovery_user ON totp_recovery_codes(user_kind, user_id);
