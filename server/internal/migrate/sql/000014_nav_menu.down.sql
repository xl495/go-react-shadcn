-- SQLite cannot cheaply drop added columns; leave them.
DROP INDEX IF EXISTS idx_nav_audience_code;
DROP TABLE IF EXISTS nav_menu;
