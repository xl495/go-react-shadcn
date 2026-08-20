# Latch · React 19 + Gin + Casbin Permission System

Graphical CAPTCHA login, JWT auth, Casbin RBAC, and SQLite persistence. The admin panel covers users, roles, permissions, dictionaries, system config, and audit logs.

## Structure

```
server/      Go 1.24 + Gin + GORM/SQLite + Casbin + JWT + CAPTCHA
admin/       React 19 + Vite + Tailwind v4 + shadcn v4 (admin console)
web/         Lightweight user-facing React app
```

### Seed Accounts

| Username | Password     | Access                                                        |
| -------- | ------------ | ------------------------------------------------------------- |
| admin    | admin123     | All menus & buttons                                           |
| operator | operator123  | User/Role/Permission/Dict/Config/Log pages, only "Create User" button |
| viewer   | viewer123    | Dashboard only                                                |

Permissions have three types: `menu` (sidebar), `button` (page actions), `api` (endpoint only). The admin UI hides buttons via `permissionCodes`; the server still enforces via Casbin.

The UI supports **Simplified Chinese / English**. Switch from the login page or sidebar; the choice is stored locally. API errors are translated by error code; seed roles/permissions are translated by code.

### Getting Started

Run server and admin concurrently:

```bash
make dev
```

Or start them separately:

```bash
cd server && go run ./cmd/server
cd admin && npm run dev             # Admin console :5173
cd web && npm run dev               # Web app :5174
```

To have the CAPTCHA endpoint return the answer (local debugging only):

```bash
CAPTCHA_DEBUG=1 go run ./cmd/server
```

Production build:

```bash
cd admin && npm run build
```

### Auth Flow

1. `GET /api/v1/auth/captcha` returns a one-time CAPTCHA (`captchaId` + base64 image).
2. `POST /api/v1/auth/login` requires username, password, and CAPTCHA to issue an HS256 JWT.
3. Protected endpoints require `Authorization: Bearer <token>`. Casbin checks username + Gin `FullPath` + HTTP method.
4. Role permissions are stored in SQLite `casbin_rule`. `PUT /api/v1/roles/:id/permissions` takes effect immediately.

### Testing

```bash
cd server && go test ./...
```

### Docker

```bash
docker compose up --build -d
```

Admin UI is on `:5173`, API on `:8080`. `GET /health` returns `{ "message": "ok" }`.

### Operations

- Schema changes live in `server/internal/migrate/sql` and run on server start.
- Mutating API calls and login attempts are written to `op_logs` and listed at `/logs`.
- `GET /metrics` exposes a Prometheus-style request counter.
