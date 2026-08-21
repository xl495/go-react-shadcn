ALTER TABLE nav_menu ADD COLUMN is_system INTEGER NOT NULL DEFAULT 0;

UPDATE nav_menu SET is_system = 1 WHERE code IN (
  'dashboard:read', 'org:menu', 'user:list', 'webuser:list', 'dept:list', 'role:list', 'perm:list',
  'system:menu', 'dict:list', 'config:list', 'mail:jobs:list', 'mail:campaign:list', 'log:list',
  'web:home', 'web:profile', 'web:password'
);
