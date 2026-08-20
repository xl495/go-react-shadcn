# Latch · React 19 + Gin + Casbin Permission System

[English](README.md) · [中文](README.zh.md)

JWT auth, Casbin RBAC, and SQLite persistence. Login can use a graphical CAPTCHA, Google reCAPTCHA, Cloudflare Turnstile, or Google sign-in — all switched from system settings. The admin panel covers users, roles, permissions, dictionaries, mail, system config, and audit logs.

## Structure

```
server/      Go 1.24 + Gin + GORM/SQLite + Casbin + JWT + captcha providers
admin/       React 19 + Vite + Tailwind v4 + shadcn v4 (admin console)
web/         Lightweight user-facing React app
```

### Seed Accounts

| Username | Password     | Access                                                                 |
| -------- | ------------ | ---------------------------------------------------------------------- |
| admin    | admin123     | All menus & buttons                                                    |
| operator | operator123  | User/Role/Permission/Dict/Config/Log pages, only "Create User" button  |
| viewer   | viewer123    | Dashboard only                                                         |
| webuser  | webuser123   | Web app only (`member` role)                                           |

Permissions have three types: `menu` (sidebar), `button` (page actions), `api` (endpoint only). The admin UI hides buttons via `permissionCodes`; the server still enforces via Casbin.

Web users cannot sign in on the admin console. The UI supports **Simplified Chinese / English**. Switch from the login page or sidebar; the choice is stored locally. API errors are translated by error code; seed roles/permissions are translated by code.

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

To have the image CAPTCHA endpoint return the answer (local debugging only):

```bash
CAPTCHA_DEBUG=1 go run ./cmd/server
```

Production build:

```bash
make build
```

### Auth Flow

1. `GET /api/v1/auth/settings` is public. It tells the login/register pages whether Google is on, which captcha provider to use, and the public site keys.
2. Password login: `POST /api/v1/auth/login` with `client` `admin` or `web`. Captcha is required unless the provider is `none`.
3. Google: the browser uses Google Identity Services to obtain an ID token, then `POST /api/v1/auth/google` with `{ idToken, client }`. The server verifies the token against `auth.google_client_id`.
4. Protected endpoints require `Authorization: Bearer <token>`. Casbin checks username + Gin `FullPath` + HTTP method.
5. Role permissions are stored in SQLite `casbin_rule`. `PUT /api/v1/roles/:id/permissions` takes effect immediately.

Forgot-password uses the same captcha provider as login.

### Sign-in settings

Admin **System settings → Sign-in**. Google and captcha keys are off by default after seed. Restart the API once after pulling this change so `auth.*` rows are inserted; if a field is still empty, fill it and save — missing keys are created on save.

#### Google

1. In Google Cloud, create an OAuth **Web application** client ID (GIS).
2. Add authorized JavaScript origins, for example `http://127.0.0.1:5173` (admin) and `http://127.0.0.1:5174` (web).
3. Paste the Client ID into **Google Client ID**, optionally store Client Secret on the server, then enable **Google sign-in**.
4. Enable **Google registration** to allow first-time Google users. Web registrations get the `member` role; admin self-register gets no extra roles. An existing email is linked when the user kind matches the client.

#### Human verification

`auth.captcha_provider`:

| Value       | Behavior |
| ----------- | -------- |
| `none`      | No captcha |
| `image`     | Built-in graphical captcha (default; `GET /api/v1/auth/captcha`) |
| `recaptcha` | reCAPTCHA v3; if the score is below `auth.recaptcha_min_score` (default `0.5`) and v2 keys exist, the UI falls back to a checkbox |
| `turnstile` | Cloudflare Turnstile |

Site keys are public. Secrets stay on the server and are redacted in the config list. `app.captcha_enabled` remains as a fallback when `auth.captcha_provider` is unset.

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
