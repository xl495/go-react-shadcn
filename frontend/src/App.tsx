import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { AppShell } from "@/components/layout/AppShell"
import { AuthProvider, RequireAuth, RequirePerm } from "@/lib/auth"
import { I18nProvider } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { ConfigsPage } from "@/pages/Configs"
import { DashboardPage } from "@/pages/Dashboard"
import { DictsPage } from "@/pages/Dicts"
import { LoginPage } from "@/pages/Login"
import { LogsPage } from "@/pages/Logs"
import { PermissionsPage } from "@/pages/Permissions"
import { RolesPage } from "@/pages/Roles"
import { SettingsPage } from "@/pages/Settings"
import { UsersPage } from "@/pages/Users"

export function App() {
  return (
    <I18nProvider>
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            element={
              <RequireAuth>
                <AppShell />
              </RequireAuth>
            }
          >
            <Route path="/" element={<DashboardPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route
              path="/users"
              element={
                <RequirePerm perm={P.userList}>
                  <UsersPage />
                </RequirePerm>
              }
            />
            <Route
              path="/roles"
              element={
                <RequirePerm perm={P.roleList}>
                  <RolesPage />
                </RequirePerm>
              }
            />
            <Route
              path="/permissions"
              element={
                <RequirePerm perm={P.permList}>
                  <PermissionsPage />
                </RequirePerm>
              }
            />
            <Route
              path="/dicts"
              element={
                <RequirePerm perm={P.dictList}>
                  <DictsPage />
                </RequirePerm>
              }
            />
            <Route
              path="/configs"
              element={
                <RequirePerm perm={P.configList}>
                  <ConfigsPage />
                </RequirePerm>
              }
            />
            <Route
              path="/logs"
              element={
                <RequirePerm perm={P.logList}>
                  <LogsPage />
                </RequirePerm>
              }
            />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
    </I18nProvider>
  )
}
