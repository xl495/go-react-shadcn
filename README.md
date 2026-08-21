# gra · React 19 + Gin + Casbin Permission System

[English](README.md) · [中文](README.zh.md)

JWT auth, Casbin RBAC, and SQLite persistence. Login can use a graphical CAPTCHA, Google reCAPTCHA, Cloudflare Turnstile, or Google sign-in — all switched from system settings. The admin panel covers users, roles, permissions, dictionaries, mail, system config, and audit logs.

## Structure

```
server/      Go 1.24 + Gin + GORM/SQLite + Casbin + JWT + captcha providers
admin/       React 19 + Vite + Tailwind v4 + shadcn v4 (admin console)
web/         Lightweight user-facing React app
```

### Seed Accounts

Development only (`APP_ENV` is not `production`). Username cannot be changed after create (Casbin subject is the user id).

| Username | Password     | Access                                                                 |
| -------- | ------------ | ---------------------------------------------------------------------- |
| admin    | admin123     | All menus & buttons                                                    |
| operator | operator123  | User/Role/Permission/Dict/Config/Log pages, only "Create User" button  |
| viewer   | viewer123    | Dashboard only                                                         |
| webuser  | webuser123   | Web app only (`member` role)                                           |

In production, set `JWT_SECRET`, a distinct `MAIL_UNSUB_SECRET`, and `BOOTSTRAP_ADMIN_PASSWORD` (upper + lower + digit, not a seed password) on first boot. Default seed passwords are rejected.

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

1. `GET /api/v1/auth/settings` is public. It tells the login/register pages whether Google is on, which captcha provider to use, public site keys, and whether email registration is enabled.
2. Password login: `POST /api/v1/auth/login` with `client` `admin` or `web`. Captcha is required unless the provider is `none`. Web password accounts created via register must verify email first (`POST /api/v1/auth/verify-email`).
3. Email register (web only): `POST /api/v1/auth/register` with captcha. The API returns `{ pending: true }` until the user opens the verification link.
4. Google: the browser uses Google Identity Services to obtain an ID token, then `POST /api/v1/auth/google` with `{ idToken, client }`. The server verifies the token against `auth.google_client_id`.
5. Protected endpoints require `Authorization: Bearer <token>`. Casbin checks `admin:{id}` / `web:{id}` + Gin `FullPath` + HTTP method. JWT includes `kind`.
6. Role permissions are stored in SQLite `casbin_rule`. `PUT /api/v1/roles/:id/permissions` takes effect immediately.
7. Navigation comes from a single `nav_menu` table (`audience` = `admin` | `web`) plus `permCode`. Permission `kind=menu` is still the RBAC catalog, not the sidebar tree.

Forgot-password uses the same captcha provider as login. Dark mode follows the system preference and can be toggled in the sidebar.

### Environment

See `.env.example`. Important variables:

| Variable | Purpose |
| -------- | ------- |
| `APP_ENV` | `production` rejects default JWT secret and seed passwords |
| `PORT` | API listen port (default `8080`) |
| `DATABASE_PATH` | SQLite file |
| `JWT_SECRET` | HS256 secret (≥32 chars in production) |
| `JWT_TTL` | Access token lifetime |
| `SESSION_CACHE_TTL` | Auth user cache (default 30s) |
| `MAIL_UNSUB_SECRET` | Unsubscribe token key, distinct from JWT |
| `BOOTSTRAP_ADMIN_PASSWORD` | First production admin password |
| `CAPTCHA_DEBUG` | Image captcha returns the answer (local only) |
| `LOG_RETENTION_DAYS` | Auto-purge audit logs |
| `API_LOG_SAMPLE` | Sample 1 of N API log writes |
| `CORS_ORIGIN` | Allowed browser origin(s) |

Ports: API `:8080`, admin `:5173`, web `:5174`. Seed passwords above are development-only; they are rejected when `APP_ENV=production`.

Health: `GET /live` (process up), `GET /ready` (SQLite ping), `GET /health` (legacy ready). `GET /metrics` is in-process and resets on restart.

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

Local Turnstile testing uses [Cloudflare dummy keys](https://developers.cloudflare.com/turnstile/troubleshooting/testing/) (widget keys, not API tokens). They work on `localhost`. Dummy sitekeys mint `XXXX.DUMMY.TOKEN.XXXX`. The dummy pass secret always verifies; the dummy fail secret always rejects.

| Site key | Secret | Result |
| --- | --- | --- |
| `1x00000000000000000000AA` (visible, always pass) | `1x0000000000000000000000000000000AA` | Login succeeds |
| `2x00000000000000000000AB` (visible, always fail) | `2x0000000000000000000000000000000AA` | Widget / verify fails |
| `1x00000000000000000000BB` (invisible, always pass) | `1x0000000000000000000000000000000AA` | Login succeeds |
| `3x00000000000000000000FF` (interactive) | `1x0000000000000000000000000000000AA` | Forces a challenge in the browser |

`3x0000000000000000000000000000000AA` returns `timeout-or-duplicate` (spent token). Production secrets reject the dummy token.

### Testing

```bash
cd server && go test ./...
```

### Docker

```bash
JWT_SECRET='replace-with-32+chars' MAIL_UNSUB_SECRET='another-32+chars' BOOTSTRAP_ADMIN_PASSWORD='YourAdmin1' docker compose up --build -d
```

Admin UI is on `:5173`, API on `:8080`. `GET /health` returns `{ "message": "ok" }`.

### Operations

- Schema changes live in `server/internal/migrate/sql` and run on server start.
- Mutating API calls and login attempts are written to `op_logs` and listed at `/logs`.
- `GET /metrics` exposes a Prometheus-style request counter.
