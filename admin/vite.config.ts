import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
      "lucide-react": path.resolve(__dirname, "node_modules/lucide-react"),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return
          if (id.includes("react-dom") || id.includes("/react/") || id.includes("scheduler")) return "react-vendor"
          if (id.includes("@tanstack")) return "query"
          if (id.includes("@radix-ui")) return "radix"
          if (id.includes("lucide-react")) return "icons"
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:8080",
      "/uploads": "http://127.0.0.1:8080",
    },
  },
})
