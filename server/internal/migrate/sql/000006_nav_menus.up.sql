CREATE TABLE IF NOT EXISTS admin_menus (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    route_path TEXT,
    component TEXT,
    icon TEXT,
    sort INTEGER NOT NULL DEFAULT 0,
    hidden INTEGER NOT NULL DEFAULT 0,
    perm_code TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME,
    updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_menus_code ON admin_menus(code);
CREATE INDEX IF NOT EXISTS idx_admin_menus_parent ON admin_menus(parent_id);
CREATE INDEX IF NOT EXISTS idx_admin_menus_perm ON admin_menus(perm_code);

CREATE TABLE IF NOT EXISTS web_menus (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    route_path TEXT,
    component TEXT,
    icon TEXT,
    sort INTEGER NOT NULL DEFAULT 0,
    hidden INTEGER NOT NULL DEFAULT 0,
    perm_code TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME,
    updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_web_menus_code ON web_menus(code);
CREATE INDEX IF NOT EXISTS idx_web_menus_parent ON web_menus(parent_id);
CREATE INDEX IF NOT EXISTS idx_web_menus_perm ON web_menus(perm_code);
