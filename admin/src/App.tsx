import { BrowserRouter, useLocation, useRoutes } from "react-router-dom"
import { NuqsAdapter } from "nuqs/adapters/react-router/v6"
import { lazy, Suspense, type ReactNode } from "react"
import { QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { useDynamicAuthRoutes } from "@/components/layout/DynamicAuthRoutes"
import { PageFallback } from "@/components/PageFallback"
import { AppShell } from "@/components/layout/AppShell"
import { AuthProvider, RequireAuth } from "@/providers/auth"
import { I18nProvider } from "@/providers/i18n"
import { queryClient } from "@/providers/query-client"

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
const ForbiddenPage = lazy(() => import("@/pages/Errors").then((m) => ({ default: m.ForbiddenPage })))
const NotFoundPage = lazy(() => import("@/pages/Errors").then((m) => ({ default: m.NotFoundPage })))

function AppRoutes() {
  const dynamicRoutes = useDynamicAuthRoutes()
  return useRoutes([
    { path: "/login", element: <LoginPage /> },
    { path: "/register", element: <RegisterPage /> },
    { path: "/forgot-password", element: <ForgotPasswordPage /> },
    { path: "/reset-password", element: <ResetPasswordPage /> },
    { path: "/unsubscribe", element: <UnsubscribePage /> },
    {
      element: (
        <RequireAuth>
          <AppShell />
        </RequireAuth>
      ),
      children: [
        ...dynamicRoutes,
        { path: "403", element: <ForbiddenPage /> },
        { path: "*", element: <NotFoundPage /> },
      ],
    },
  ])
}

function ResettableErrorBoundary({ children }: { children: ReactNode }) {
  const location = useLocation()
  return <ErrorBoundary resetKey={location.key}>{children}</ErrorBoundary>
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <AuthProvider>
          <BrowserRouter>
            <NuqsAdapter>
              <ResettableErrorBoundary>
                <Toaster position="top-center" richColors />
                <Suspense fallback={<PageFallback />}>
                  <AppRoutes />
                </Suspense>
              </ResettableErrorBoundary>
            </NuqsAdapter>
          </BrowserRouter>
        </AuthProvider>
      </I18nProvider>
    </QueryClientProvider>
  )
}
