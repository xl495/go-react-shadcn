export { ApiError, getToken, setToken, setUnauthorizedHandler } from "./http"
import { authApi } from "./auth"
import { usersApi } from "./users"
import { rolesApi } from "./roles"
import { permissionsApi } from "./permissions"
import { dictsApi } from "./dicts"
import { configsApi } from "./configs"
import { logsApi } from "./logs"
import { departmentsApi } from "./departments"

export const api = {
  ...authApi,
  ...usersApi,
  ...rolesApi,
  ...permissionsApi,
  ...dictsApi,
  ...configsApi,
  ...logsApi,
  ...departmentsApi,
}
