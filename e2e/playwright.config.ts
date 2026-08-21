import { defineConfig } from "@playwright/test"

const adminURL = process.env.E2E_ADMIN_URL || "http://127.0.0.1:5173"

export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  use: {
    baseURL: adminURL,
    trace: "retain-on-failure",
  },
})
