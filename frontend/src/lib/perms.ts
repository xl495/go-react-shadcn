export const P = {
  dashboard: "dashboard:read",
  userList: "user:list",
  userCreate: "user:create",
  userUpdate: "user:update",
  userDelete: "user:delete",
  userRoles: "user:roles",
  userAvatar: "user:avatar",
  roleList: "role:list",
  roleCreate: "role:create",
  roleUpdate: "role:update",
  roleDelete: "role:delete",
  rolePerms: "role:perms",
  permList: "perm:list",
  permCreate: "perm:create",
  permUpdate: "perm:update",
  permDelete: "perm:delete",
  dictList: "dict:list",
  dictCreate: "dict:create",
  dictUpdate: "dict:update",
  dictDelete: "dict:delete",
  dictItemCreate: "dict:item:create",
  dictItemUpdate: "dict:item:update",
  dictItemDelete: "dict:item:delete",
  configList: "config:list",
  configCreate: "config:create",
  configUpdate: "config:update",
  configDelete: "config:delete",
  logList: "log:list",
  logClear: "log:clear",
} as const

export type PermCode = (typeof P)[keyof typeof P]

export const KIND_LABEL: Record<string, string> = {
  menu: "菜单",
  button: "按钮",
  api: "接口",
}
