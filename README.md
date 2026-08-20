# Latch · React 19 + Gin + Casbin Permission System

Graphical CAPTCHA login, JWT auth, Casbin RBAC, and SQLite persistence. The admin panel covers users, roles, permissions, dictionaries, system config, and audit logs.

## Structure

```
backend/     Go 1.24 + Gin + GORM/SQLite + Casbin + JWT + CAPTCHA
frontend/    React 19 + Vite + Tailwind v4 + shadcn v4
```

### Seed Accounts

| Username | Password     | Access                                                        |
| -------- | ------------ | ------------------------------------------------------------- |
| admin    | admin123     | All menus & buttons                                           |
| operator | operator123  | User/Role/Permission/Dict/Config/Log pages, only "Create User" button |
| viewer   | viewer123    | Dashboard only                                                |

Permissions have three types: `menu` (sidebar), `button` (page actions), `api` (endpoint only). The frontend hides buttons via `permissionCodes`; the backend still enforces via Casbin.

The UI supports **Simplified Chinese / Traditional Chinese / English**. Switch from the login page or sidebar; the choice is stored locally. API errors are translated by error code; seed roles/permissions are translated by code.

### Getting Started

Run both frontend and backend concurrently:

```bash
make dev
```

Or start them separately:

```bash
cd backend && go run ./cmd/server
cd frontend && npm run dev          # Admin panel :5173
cd web && npm run dev               # Web app :5174
```

To have the CAPTCHA endpoint return the answer (local debugging only):

```bash
CAPTCHA_DEBUG=1 go run ./cmd/server
```

Production build:

```bash
cd frontend && npm run build
```

### Auth Flow

1. `GET /api/v1/auth/captcha` returns a one-time CAPTCHA (`captchaId` + base64 image).
2. `POST /api/v1/auth/login` requires username, password, and CAPTCHA to issue an HS256 JWT.
3. Protected endpoints require `Authorization: Bearer <token>`. Casbin checks username + Gin `FullPath` + HTTP method.
4. Role permissions are stored in SQLite `casbin_rule`. `PUT /api/v1/roles/:id/permissions` takes effect immediately.

### Testing

```bash
cd backend && go test ./...
```

### Docker

```bash
docker compose up --build -d
```

Admin UI is on `:5173`, API on `:8080`. `GET /health` returns `{ "message": "ok" }`.

### Operations

- Schema changes live in `backend/internal/migrate/sql` and run on server start.
- Mutating API calls and login attempts are written to `op_logs` and listed at `/logs`.
- `GET /metrics` exposes a Prometheus-style request counter.
