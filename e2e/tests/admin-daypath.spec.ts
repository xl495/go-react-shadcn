import { expect, test } from "@playwright/test"
import { loginAdmin } from "./helpers"

test.beforeEach(async ({ page }) => {
  await loginAdmin(page)
})

test("deep links to users and role detail do not 404", async ({ page }) => {
  await page.goto("/users")
  await expect(page).not.toHaveURL(/\/login$/)
  await expect(page.getByRole("heading", { name: /staff users|后台用户/i })).toBeVisible()
  await expect(page.getByText("admin").first()).toBeVisible()

  await page.goto("/roles")
  const detail = page.getByRole("link", { name: /details|详情/i }).first()
  await expect(detail).toBeVisible()
  await detail.click()
  await expect(page).toHaveURL(/\/roles\/\d+/)
  await expect(page.getByText(/page not found|页面不存在/i)).toHaveCount(0)
  await expect(page.locator("#role-name")).toBeVisible()

  const roleURL = page.url()
  await page.goto("/")
  await page.goto(roleURL)
  await expect(page).toHaveURL(/\/roles\/\d+/)
  await expect(page.getByText(/page not found|页面不存在/i)).toHaveCount(0)
  await expect(page.locator("#role-name")).toBeVisible()
})

test("dashboard shows numeric stats after load", async ({ page }) => {
  await page.goto("/")
  await expect(page.getByRole("heading", { name: /workbench|仪表盘/i })).toBeVisible()
  await expect(page.locator(".tabular-nums").first()).toHaveText(/\d+/, { timeout: 15_000 })
})

test("sidebar and breadcrumbs expose i18n landmarks", async ({ page }) => {
  await page.goto("/")
  await expect(page.getByRole("navigation", { name: /main navigation|主导航/i })).toBeVisible()
  await page.goto("/users")
  await expect(page.getByRole("navigation", { name: /breadcrumb|面包屑/i })).toBeVisible()
})

test("google and smtp extra fields stay hidden until enabled", async ({ page }) => {
  await page.goto("/configs?tab=auth")
  await expect(page.getByRole("tab", { name: /sign-in|登录/i })).toHaveAttribute("aria-selected", "true")
  await expect(page.locator("#auth-google_client_id")).toHaveCount(0)
  await page.locator("#auth-google_enabled").click()
  await expect(page.locator("#auth-google_client_id")).toBeVisible()

  await page.goto("/configs?tab=mail&section=smtp")
  await expect(page.locator("#mail-host")).toHaveCount(0)
  await page.locator("#mail-enabled").click()
  await expect(page.locator("#mail-host")).toBeVisible()
})
