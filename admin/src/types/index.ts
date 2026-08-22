import type { components } from "./generated/api-schema"

type Schema = components["schemas"]

export type User = Schema["User"] & {
  totpEnabled?: boolean
  lockedUntil?: string | null
  mustChangePassword?: boolean
  googleBound?: boolean
  pendingEmail?: string
  emailVerifyToken?: string
}
export type Role = Schema["Role"] & { permissionIds?: number[] }
export type Permission = Schema["Permission"]
export type MenuNode = Schema["MenuNode"]
export type OpLog = Schema["OpLog"]
export type LoginLog = Schema["LoginLog"]
export type APILog = Schema["APILog"]
export type Department = Schema["Department"]
export type DictType = Schema["DictType"]
export type DictItem = Schema["DictItem"]
export type SysConfig = Schema["SysConfig"]
export type MailJob = Schema["MailJob"]
export type MailCampaign = Schema["MailCampaign"]
export type DashboardStats = Schema["DashboardStats"]
export type CaptchaChallenge = Schema["CaptchaChallenge"]
export type LoginResult = Schema["LoginResult"] & {
  token?: string
  totpRequired?: boolean
  totpTicket?: string
  totpEnroll?: boolean
  recoveryCodes?: string[]
  user?: User
}
export type AuthSettings = Schema["AuthSettings"] & { maintenance?: boolean }
export type DictLookup = Schema["DictLookup"]

export type PageResult<T> = {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export function emptyPage<T>(page = 1, pageSize = 10): PageResult<T> {
  return { items: [], total: 0, page, pageSize }
}
