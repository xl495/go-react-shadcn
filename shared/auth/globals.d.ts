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

interface Window {
  grecaptcha?: Grecaptcha
  turnstile?: TurnstileApi
}
