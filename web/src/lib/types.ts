export type User = {
  id: number
  username: string
  nickname: string
  avatar: string
  email?: string
  pendingEmail?: string
  phone: string
  gender?: string
  department?: string
  title?: string
  remark?: string
  status?: string
  timezone?: string
  marketingOptIn?: boolean
  totpEnabled?: boolean
  mustChangePassword?: boolean
  mustSetPassword?: boolean
  googleBound?: boolean
}

export type LoginResult = {
  token?: string
  expiresAt?: string
  user?: User
  totpRequired?: boolean
  totpTicket?: string
  totpEnroll?: boolean
  recoveryCodes?: string[]
}

export type CaptchaChallenge = {
  captchaId: string
  image: string
}

export type AuthSettings = {
  googleEnabled: boolean
  googleRegisterEnabled: boolean
  registerEnabled?: boolean
  googleClientId: string
  captchaProvider: string
  recaptchaSiteKeyV3?: string
  recaptchaSiteKeyV2?: string
  turnstileSiteKey?: string
  maintenance?: boolean
}

export type MenuNode = {
  id: number
  name: string
  code: string
  kind: string
  routePath: string
  component: string
  icon: string
  sort: number
  hidden: boolean
  permCode?: string
  children?: MenuNode[]
}
