import { expect, type Page } from "@playwright/test"

export const webURL = process.env.E2E_WEB_URL || "http://127.0.0.1:5174"

export async function loginAdmin(page: Page) {
  let captcha: { captchaId?: string; answer?: string } = {}
  await page.route("**/api/v1/auth/captcha", async (route) => {
    const res = await route.fetch()
    const body = await res.json()
    captcha = body.data ?? body
    await route.fulfill({ response: res, json: body })
  })

  await page.goto("/login")
  await page.getByLabel(/username|用户名/i).fill("admin")
  await page.locator("#password").fill("admin123")
  await expect.poll(() => captcha.answer || "").not.toEqual("")
  const captchaInput = page.locator("#captcha")
  if (await captchaInput.count()) {
    await captchaInput.fill(captcha.answer || "")
  }
  await page.getByRole("button", { name: /sign in|登录/i }).click()
  await expect(page).not.toHaveURL(/\/login$/)
}
