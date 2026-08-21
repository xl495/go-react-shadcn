import { expect, test } from "@playwright/test"
import { loginAdmin } from "./helpers"

test("admin login reaches dashboard and user list", async ({ page }) => {
  await loginAdmin(page)
  await page.goto("/users")
  await expect(page.getByRole("heading").first()).toBeVisible()
  await expect(page.getByText("admin").first()).toBeVisible()
})
