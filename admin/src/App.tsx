import { createBrowserRouter, Outlet, RouterProvider, useLocation } from "react-router-dom"
import { NuqsAdapter } from "nuqs/adapters/react-router/v7"
import { lazy, Suspense } from "react"
import { QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { buildAuthRoutes } from "@/components/layout/DynamicAuthRoutes"
import { PageFallback } from "@/components/PageFallback"
import { AppShell } from "@/components/layout/AppShell"
import { AuthProvider, RequireAuth } from "@/providers/auth"
import { I18nProvider } from "@/providers/i18n"
import { ThemeProvider, useTheme } from "@/providers/theme"
import { queryClient } from "@/providers/query-client"
import { FALLBACK_MENU_ROUTES } from "@/hooks/queries"

const LoginPage = lazy(() => import("@/pages/Login").then((m) => ({ default: m.LoginPage })))
const RegisterPage = lazy(() => import("@/pages/Register").then((m) => ({ default: m.RegisterPage })))
const ForgotPasswordPage = lazy(() =>
  import("@/pages/ForgotPassword").then((m) => ({ default: m.ForgotPasswordPage })),
)
const ResetPasswordPage = lazy(() =>
  import("@/pages/ForgotPassword").then((m) => ({ default: m.ResetPasswordPage })),
)
const UnsubscribePage = lazy(() =>
  import("@/pages/Unsubscribe").then((m) => ({ default: m.UnsubscribePage })),
)
const VerifyEmailPage = lazy(() =>
  import("@/pages/VerifyEmail").then((m) => ({ default: m.VerifyEmailPage })),
)
const ForbiddenPage = lazy(() => import("@/pages/Errors").then((m) => ({ default: m.ForbiddenPage })))
const NotFoundPage = lazy(() => import("@/pages/Errors").then((m) => ({ default: m.NotFoundPage })))

function AppFrame() {
  const location = useLocation()
  const { resolved } = useTheme()
  return (
    <NuqsAdapter>
      <ErrorBoundary resetKey={location.key}>
        <Toaster position="top-center" richColors theme={resolved} />
        <Suspense fallback={<PageFallback />}>
          <Outlet />
        </Suspense>
      </ErrorBoundary>
    </NuqsAdapter>
  )
}

const router = createBrowserRouter([
  {
    element: <AppFrame />,
    children: [
      { path: "login", element: <LoginPage /> },
      { path: "register", element: <RegisterPage /> },
      { path: "forgot-password", element: <ForgotPasswordPage /> },
      { path: "reset-password", element: <ResetPasswordPage /> },
      { path: "unsubscribe", element: <UnsubscribePage /> },
      { path: "verify-email", element: <VerifyEmailPage /> },
      {
        element: (
          <RequireAuth>
            <AppShell />
          </RequireAuth>
        ),
        children: [
          ...buildAuthRoutes(FALLBACK_MENU_ROUTES),
          { path: "403", element: <ForbiddenPage /> },
          { path: "*", element: <NotFoundPage /> },
        ],
      },
    ],
  },
])

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <I18nProvider>
          <AuthProvider>
            <RouterProvider router={router} />
          </AuthProvider>
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}
