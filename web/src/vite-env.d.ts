/// <reference types="vite/client" />

interface Grecaptcha {
  ready(cb: () => void): void
  execute(siteKey: string, opts: { action: string }): Promise<string>
  render(
    el: HTMLElement,
    opts: {
      sitekey: string
      callback: (token: string) => void
      "expired-callback"?: () => void
    },
  ): number
  reset(id?: number): void
}

interface TurnstileApi {
  render(
    el: HTMLElement,
    opts: {
      sitekey: string
      callback: (token: string) => void
      "expired-callback"?: () => void
      theme?: string
    },
  ): string
  reset(id: string): void
  remove(id: string): void
}

interface GoogleAccountsId {
  initialize(opts: {
    client_id: string
    callback: (res: { credential: string }) => void
    ux_mode?: string
  }): void
  renderButton(
    el: HTMLElement,
    opts: { theme?: string; size?: string; width?: number; text?: string; locale?: string },
  ): void
}

interface Window {
  grecaptcha?: Grecaptcha
  turnstile?: TurnstileApi
  google?: { accounts: { id: GoogleAccountsId } }
}
