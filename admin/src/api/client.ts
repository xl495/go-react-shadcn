export { ApiError, getToken, setToken, setUnauthorizedHandler, isSessionExpired } from "./http"
import { authApi } from "./auth"
import { usersApi } from "./users"
import { rolesApi } from "./roles"
import { permissionsApi } from "./permissions"
import { dictsApi } from "./dicts"
import { configsApi } from "./configs"
import { logsApi } from "./logs"
import { departmentsApi } from "./departments"
import { mailApi } from "./mail"
import { menusApi, notificationsApi } from "./menus"
import { totpApi } from "./totp"

export const api = {
  ...authApi,
  ...usersApi,
  ...rolesApi,
  ...permissionsApi,
  ...dictsApi,
  ...configsApi,
  ...logsApi,
  ...departmentsApi,
  ...mailApi,
  ...menusApi,
  ...notificationsApi,
  ...totpApi,
}
