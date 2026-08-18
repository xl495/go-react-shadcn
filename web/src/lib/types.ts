export type User = {
  id: number
  username: string
  nickname: string
  avatar: string
  email?: string
  phone: string
  status?: string
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
