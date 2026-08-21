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
    created_at DATETIME,
    updated_at DATETIME
);

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
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS web_user_roles (
    user_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    PRIMARY KEY (user_id, role_id)
);

INSERT INTO admin_user (
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    timezone, marketing_opt_in, google_id, created_at, updated_at
)
SELECT
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    COALESCE(timezone, 'Asia/Shanghai'), COALESCE(marketing_opt_in, 1), COALESCE(google_id, ''), created_at, updated_at
FROM users
WHERE COALESCE(kind, '') IN ('', 'admin');

INSERT INTO web_user (
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    timezone, marketing_opt_in, google_id, created_at, updated_at
)
SELECT
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    COALESCE(timezone, 'Asia/Shanghai'), COALESCE(marketing_opt_in, 1), COALESCE(google_id, ''), created_at, updated_at
FROM users
WHERE kind = 'web';

INSERT INTO web_user_roles (user_id, role_id)
SELECT ur.user_id, ur.role_id
FROM user_roles ur
WHERE ur.user_id IN (SELECT id FROM web_user);

DELETE FROM user_roles
WHERE user_id IN (SELECT id FROM web_user);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_user_email_lower ON admin_user(lower(email)) WHERE email <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_web_user_email_lower ON web_user(lower(email)) WHERE email <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_user_google_id_nonempty ON admin_user(google_id) WHERE google_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_web_user_google_id_nonempty ON web_user(google_id) WHERE google_id <> '';
CREATE INDEX IF NOT EXISTS idx_admin_user_department_id ON admin_user(department_id);
CREATE INDEX IF NOT EXISTS idx_web_user_department_id ON web_user(department_id);
CREATE INDEX IF NOT EXISTS idx_admin_user_google_id ON admin_user(google_id);
CREATE INDEX IF NOT EXISTS idx_web_user_google_id ON web_user(google_id);
CREATE INDEX IF NOT EXISTS idx_web_user_roles_role_id ON web_user_roles(role_id);

ALTER TABLE password_reset_tokens ADD COLUMN user_kind TEXT NOT NULL DEFAULT 'admin';
ALTER TABLE mail_jobs ADD COLUMN user_kind TEXT NOT NULL DEFAULT '';
