DELETE FROM user_roles WHERE user_id IN (SELECT id FROM web_user);
INSERT INTO user_roles (user_id, role_id)
SELECT user_id, role_id FROM web_user_roles
WHERE user_id NOT IN (SELECT user_id FROM user_roles);

INSERT INTO users (
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    timezone, marketing_opt_in, google_id, kind, created_at, updated_at
)
SELECT
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    timezone, marketing_opt_in, google_id, 'admin', created_at, updated_at
FROM admin_user
WHERE id NOT IN (SELECT id FROM users);

INSERT INTO users (
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    timezone, marketing_opt_in, google_id, kind, created_at, updated_at
)
SELECT
    id, username, password_hash, nickname, avatar, email, phone, gender, department, department_id,
    title, remark, status, token_version, failed_login_count, locked_until, last_login_at, last_login_ip,
    timezone, marketing_opt_in, google_id, 'web', created_at, updated_at
FROM web_user
WHERE id NOT IN (SELECT id FROM users);

DROP TABLE IF EXISTS web_user_roles;
DROP TABLE IF EXISTS admin_user;
DROP TABLE IF EXISTS web_user;
