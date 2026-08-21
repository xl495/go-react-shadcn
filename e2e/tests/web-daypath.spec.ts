import { expect, test } from "@playwright/test"
import { webURL } from "./helpers"

test("unknown web path shows a dedicated 404 page", async ({ page }) => {
  await page.goto(`${webURL}/this-page-does-not-exist`)
  await expect(page).not.toHaveURL(new RegExp(`${webURL.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/?$`))
  await expect(page.getByText("404")).toBeVisible()
  await expect(page.getByRole("heading", { name: /page not found|页面不存在/i })).toBeVisible()
  await expect(page.getByRole("link", { name: /go home|回到首页/i })).toBeVisible()
})

test("web login forgot and register use in-app links", async ({ page }) => {
  await page.goto(`${webURL}/login`)
  const forgot = page.getByRole("link", { name: /forgot password|忘记密码/i })
  await expect(forgot).toHaveAttribute("href", "/forgot-password")
  await forgot.click()
  await expect(page).toHaveURL(/\/forgot-password$/)
  await expect(page.getByRole("heading", { name: /reset password|找回密码/i })).toBeVisible()

  await page.goto(`${webURL}/login`)
  const register = page.getByRole("link", { name: /^register$|^注册$/i })
  if (await register.count()) {
    await expect(register).toHaveAttribute("href", "/register")
    await register.click()
    await expect(page).toHaveURL(/\/register$/)
  }
})
