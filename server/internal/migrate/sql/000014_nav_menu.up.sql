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
  created_at DATETIME,
  updated_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_nav_audience_code ON nav_menu(audience, code);

INSERT INTO nav_menu (audience, name, code, route_path, component, icon, sort, hidden, perm_code, status, created_at, updated_at)
SELECT 'admin', name, code, route_path, component, icon, sort, hidden, perm_code, status, created_at, updated_at
FROM admin_menus
WHERE NOT EXISTS (SELECT 1 FROM nav_menu n WHERE n.audience = 'admin' AND n.code = admin_menus.code);

INSERT INTO nav_menu (audience, name, code, route_path, component, icon, sort, hidden, perm_code, status, created_at, updated_at)
SELECT 'web', name, code, route_path, component, icon, sort, hidden, perm_code, status, created_at, updated_at
FROM web_menus
WHERE NOT EXISTS (SELECT 1 FROM nav_menu n WHERE n.audience = 'web' AND n.code = web_menus.code);

UPDATE nav_menu
SET parent_id = (
  SELECT p.id
  FROM nav_menu p
  JOIN admin_menus child ON child.code = nav_menu.code
  JOIN admin_menus parent ON parent.id = child.parent_id
  WHERE nav_menu.audience = 'admin' AND p.audience = 'admin' AND p.code = parent.code
)
WHERE audience = 'admin';

UPDATE nav_menu
SET parent_id = (
  SELECT p.id
  FROM nav_menu p
  JOIN web_menus child ON child.code = nav_menu.code
  JOIN web_menus parent ON parent.id = child.parent_id
  WHERE nav_menu.audience = 'web' AND p.audience = 'web' AND p.code = parent.code
)
WHERE audience = 'web';

ALTER TABLE admin_user ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 1;
ALTER TABLE web_user ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 1;
ALTER TABLE password_reset_tokens ADD COLUMN purpose TEXT NOT NULL DEFAULT 'reset';
