export type User = {
  id: number
  username: string
  nickname: string
  avatar: string
  email?: string
  phone: string
  gender?: string
  department?: string
  title?: string
  remark?: string
  status?: string
  timezone?: string
  marketingOptIn?: boolean
}

export type LoginResult = {
  token: string
  expiresAt: string
  user: User
}

export type CaptchaChallenge = {
  captchaId: string
  image: string
}
