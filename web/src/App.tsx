import { BrowserRouter, Route, Routes, useLocation } from "react-router-dom"
import { lazy, Suspense, type ReactNode } from "react"
import { AuthProvider, RequireAuth, RequireGuest } from "@/lib/auth"
import { I18nProvider, useI18n } from "@/lib/i18n"
import { ThemeProvider } from "@/providers/theme"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { AppShell } from "@/components/AppShell"

const HomePage = lazy(() => import("@/pages/Home").then((m) => ({ default: m.HomePage })))
const LoginPage = lazy(() => import("@/pages/Login").then((m) => ({ default: m.LoginPage })))
const RegisterPage = lazy(() => import("@/pages/Register").then((m) => ({ default: m.RegisterPage })))
const ForgotPasswordPage = lazy(() =>
  import("@/pages/ForgotPassword").then((m) => ({ default: m.ForgotPasswordPage })),
)
const ResetPasswordPage = lazy(() =>
  import("@/pages/ForgotPassword").then((m) => ({ default: m.ResetPasswordPage })),
)
const VerifyEmailPage = lazy(() => import("@/pages/VerifyEmail").then((m) => ({ default: m.VerifyEmailPage })))
const UnsubscribePage = lazy(() => import("@/pages/Unsubscribe").then((m) => ({ default: m.UnsubscribePage })))
const ProfilePage = lazy(() => import("@/pages/Profile").then((m) => ({ default: m.ProfilePage })))
const PasswordPage = lazy(() => import("@/pages/Password").then((m) => ({ default: m.PasswordPage })))
const NotFoundPage = lazy(() => import("@/pages/NotFound").then((m) => ({ default: m.NotFoundPage })))

function ResettableErrorBoundary({ children }: { children: ReactNode }) {
  const location = useLocation()
  return <ErrorBoundary resetKey={location.key}>{children}</ErrorBoundary>
}

function PageFallback() {
  const { t } = useI18n()
  return <p className="p-8 text-sm text-muted-foreground">{t("app.loading")}</p>
}

export function App() {
  return (
    <I18nProvider>
      <ThemeProvider>
        <AuthProvider>
          <BrowserRouter>
            <ResettableErrorBoundary>
              <Suspense fallback={<PageFallback />}>
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
                  <Route path="/verify-email" element={<VerifyEmailPage />} />
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
                  <Route path="*" element={<NotFoundPage />} />
                </Routes>
              </Suspense>
            </ResettableErrorBoundary>
          </BrowserRouter>
        </AuthProvider>
      </ThemeProvider>
    </I18nProvider>
  )
}
