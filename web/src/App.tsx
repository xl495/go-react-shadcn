import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { AuthProvider, RequireAuth, RequireGuest } from "@/lib/auth"
import { AppShell } from "@/components/AppShell"
import { HomePage } from "@/pages/Home"
import { LoginPage } from "@/pages/Login"
import { RegisterPage } from "@/pages/Register"
import { ForgotPasswordPage, ResetPasswordPage } from "@/pages/ForgotPassword"
import { UnsubscribePage } from "@/pages/Unsubscribe"
import { ProfilePage } from "@/pages/Profile"
import { PasswordPage } from "@/pages/Password"

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
            path="/register"
            element={
              <RequireGuest>
                <RegisterPage />
              </RequireGuest>
            }
          />
          <Route
            path="/forgot-password"
            element={
              <RequireGuest>
                <ForgotPasswordPage />
              </RequireGuest>
            }
          />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/unsubscribe" element={<UnsubscribePage />} />
          <Route
            element={
              <RequireAuth>
                <AppShell />
              </RequireAuth>
            }
          >
            <Route path="/" element={<HomePage />} />
            <Route path="/profile" element={<ProfilePage />} />
            <Route path="/password" element={<PasswordPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
