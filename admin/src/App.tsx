import { BrowserRouter, Route, Routes } from "react-router-dom"
import { lazy, Suspense } from "react"
import { QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { DynamicAuthRoutes } from "@/components/layout/DynamicAuthRoutes"
import { PageFallback } from "@/components/PageFallback"
import { AppShell } from "@/components/layout/AppShell"
import { AuthProvider, RequireAuth } from "@/providers/auth"
import { I18nProvider } from "@/providers/i18n"
import { queryClient } from "@/providers/query-client"

const LoginPage = lazy(() => import("@/pages/Login").then((m) => ({ default: m.LoginPage })))
const ForbiddenPage = lazy(() => import("@/pages/Errors").then((m) => ({ default: m.ForbiddenPage })))
const NotFoundPage = lazy(() => import("@/pages/Errors").then((m) => ({ default: m.NotFoundPage })))

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <AuthProvider>
          <ErrorBoundary>
            <Toaster position="top-center" richColors />
            <BrowserRouter>
              <Suspense fallback={<PageFallback />}>
                <Routes>
                  <Route path="/login" element={<LoginPage />} />
                  <Route
                    element={
                      <RequireAuth>
                        <AppShell />
                      </RequireAuth>
                    }
                  >
                    <DynamicAuthRoutes />
                    <Route path="403" element={<ForbiddenPage />} />
                    <Route path="*" element={<NotFoundPage />} />
                  </Route>
                </Routes>
              </Suspense>
            </BrowserRouter>
          </ErrorBoundary>
        </AuthProvider>
      </I18nProvider>
    </QueryClientProvider>
  )
}
