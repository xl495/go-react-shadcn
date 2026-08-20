DROP INDEX IF EXISTS idx_op_logs_trace_id;
-- SQLite cannot drop columns easily; downgrade recreates from backup in dev only.
DROP INDEX IF EXISTS idx_api_logs_created_at;
DROP INDEX IF EXISTS idx_api_logs_trace_id;
DROP TABLE IF EXISTS api_logs;
DROP INDEX IF EXISTS idx_login_logs_created_at;
DROP INDEX IF EXISTS idx_login_logs_username;
DROP TABLE IF EXISTS login_logs;
DROP INDEX IF EXISTS idx_departments_parent_id;
DROP TABLE IF EXISTS departments;
