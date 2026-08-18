export type Permission = {
  id: number
  name: string
  code: string
  path: string
  method: string
  kind: string
  description: string
}

export type Role = {
  id: number
  name: string
  code: string
  description: string
  permissions?: Permission[]
}

export type User = {
  id: number
  username: string
  nickname: string
  avatar: string
  email: string
  phone: string
  gender: string
  department: string
  title: string
  remark: string
  status: string
  lastLoginAt?: string | null
  lastLoginIp?: string
  roles: Role[]
  permissionCodes?: string[]
  createdAt?: string
  updatedAt?: string
}

export type LoginResult = {
  token: string
  expiresAt: string
  user: User
}

export type CaptchaChallenge = {
  captchaId: string
  image: string
  answer?: string
}

export type DashboardStats = {
  users: number
  roles: number
  permissions: number
  dicts: number
  configs: number
  logs: number
}

export type DictType = {
  id: number
  code: string
  name: string
  status: string
  remark: string
}

export type DictItem = {
  id: number
  typeCode: string
  label: string
  value: string
  sort: number
  status: string
  remark: string
}

export type SysConfig = {
  id: number
  key: string
  value: string
  name: string
  group: string
  remark: string
}

export type OpLog = {
  id: number
  username: string
  module: string
  action: string
  method: string
  path: string
  status: number
  ip: string
  latencyMs: number
  detail: string
  createdAt: string
}
