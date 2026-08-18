import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { AuthProvider, RequireAuth, RequireGuest } from "@/lib/auth"
import { HomePage } from "@/pages/Home"
import { LoginPage } from "@/pages/Login"

export function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route
            path="/login"
            element={
              <RequireGuest>
                <LoginPage />
              </RequireGuest>
            }
          />
          <Route
            path="/"
            element={
              <RequireAuth>
                <HomePage />
              </RequireAuth>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
