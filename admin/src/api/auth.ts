import { request, uploadAvatar } from "./http"
import type { CaptchaChallenge, LoginResult, MenuNode, User } from "@/types"

export const authApi = {
  captcha: () => request<CaptchaChallenge>("/api/v1/auth/captcha"),
  login: (body: {
    username: string
    password: string
    captchaId: string
    captchaCode: string
  }) =>
    request<LoginResult>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  me: () => request<User>("/api/v1/auth/me"),
  menus: () => request<MenuNode[]>("/api/v1/auth/menus"),
  updateProfile: (body: {
    nickname: string
    email: string
    phone: string
    gender: string
    department: string
    title: string
    remark: string
  }) => request<User>("/api/v1/auth/profile", { method: "PUT", body: JSON.stringify(body) }),
  uploadOwnAvatar: (file: File) => uploadAvatar("/api/v1/auth/avatar", file),
  changePassword: (body: { oldPassword: string; newPassword: string }) =>
    request<{ changed: boolean }>("/api/v1/auth/password", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
}
