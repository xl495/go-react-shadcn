CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users(lower(email)) WHERE email <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id_nonempty ON users(google_id) WHERE google_id <> '';
CREATE INDEX IF NOT EXISTS idx_users_department_id ON users(department_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);
