# gra · React 19 + Gin + Casbin 登录权限系统

[English](README.md) · [中文](README.zh.md)

JWT 认证、Casbin RBAC、SQLite 持久化。登录可使用图形验证码、Google reCAPTCHA、Cloudflare Turnstile 或 Google 账号，均在系统设置中切换。管理端覆盖用户、角色、权限、字典、邮件、系统参数与审计日志。

## 中文

### 目录

```
server/      Go 1.24 + Gin + GORM/SQLite + Casbin + JWT + 多种人机验证
admin/       React 19 + Vite + Tailwind v4 + shadcn v4（管理端）
web/         面向用户的轻量 React 应用
```

### 种子账号

仅开发环境（`APP_ENV` 不是 `production`）。创建后不可改用户名（Casbin 主体是用户 id）。

| 账号     | 密码         | 权限                                              |
| -------- | ------------ | ------------------------------------------------- |
| admin    | admin123     | 全部菜单与按钮                                    |
| operator | operator123  | 能进用户/角色/权限/字典/参数/日志页，但只有「新建用户」按钮 |
| viewer   | viewer123    | 仅仪表盘                                          |
| webuser  | webuser123   | 仅用户端（`member` 角色）                         |

生产环境首次启动需设置 `JWT_SECRET`、与之不同的 `MAIL_UNSUB_SECRET`，以及 `BOOTSTRAP_ADMIN_PASSWORD`（大小写+数字，不能是种子口令）。默认种子密码会被拒绝。

权限分三类：`menu`（侧栏）、`button`（页面按钮）、`api`（纯接口）。前端按 `permissionCodes` 隐藏按钮，后端 Casbin 仍拦截对应 HTTP 方法。

用户端账号不能登录管理端。界面支持 **简体中文 / English**。登录页右上角和侧栏可切换，选择会记在本地；接口错误按错误码翻译，种子角色/权限按编码翻译。

### 启动

前后端并发：

```bash
make dev
```

分别启动：

```bash
cd server && go run ./cmd/server
cd admin && npm run dev             # 管理端 :5173
cd web && npm run dev               # Web 端 :5174
```

调试登录时让图形验证码接口回传答案（仅本地验证用）：

```bash
CAPTCHA_DEBUG=1 go run ./cmd/server
```

生产构建：

```bash
make build
```

### 认证流程

1. `GET /api/v1/auth/settings` 公开可读，供登录/注册页判断是否开启 Google、使用哪种验证码、站点公钥，以及是否开放邮箱注册。
2. 密码登录：`POST /api/v1/auth/login`，`client` 为 `admin` 或 `web`。验证码提供方不是 `none` 时必须过人机验证。用户端自助注册账号需先 `POST /api/v1/auth/verify-email`。
3. 邮箱注册（仅 web）：`POST /api/v1/auth/register`，带验证码。接口返回 `{ pending: true }`，点邮件链接后才可登录。
4. Google：浏览器通过 Google Identity Services 拿到 ID token，再 `POST /api/v1/auth/google`，body 为 `{ idToken, client }`。服务端按 `auth.google_client_id` 校验 token。
5. 受保护接口带 `Authorization: Bearer <token>`。Casbin 用 `admin:{id}` / `web:{id}` + Gin `FullPath` + Method 判定。JWT 含 `kind`。
6. 角色权限写入 SQLite `casbin_rule`，`PUT /api/v1/roles/:id/permissions` 后立即改变 Enforce 结果。
7. 侧栏来自单一 `nav_menu` 表（`audience` = admin/web）加 `permCode`。权限 `kind=menu` 仍是 RBAC 目录，不是导航树。

忘记密码与登录使用同一套人机验证。暗色模式跟随系统，也可在侧栏切换。

种子密码仅开发环境使用；`APP_ENV=production` 时会被拒绝。健康检查：`GET /live` 进程存活，`GET /ready` 数据库就绪。

### 环境变量

见 `.env.example`。常用项：`JWT_SECRET`、`MAIL_UNSUB_SECRET`、`BOOTSTRAP_ADMIN_PASSWORD`、`SESSION_CACHE_TTL`、`CAPTCHA_DEBUG`、`LOG_RETENTION_DAYS`。端口：API `:8080`，管理端 `:5173`，用户端 `:5174`。

### 登录与验证设置

管理端 **系统设置 → 登录**。种子数据里 Google 与第三方验证码默认关闭。拉代码后请重启一次 API，以便写入 `auth.*` 配置；若表单项仍为空，填好后保存即可创建缺失的键。

#### Google

1. 在 Google Cloud 创建 OAuth **Web application** 客户端 ID（GIS）。
2. 配置授权 JavaScript 来源，例如 `http://127.0.0.1:5173`（管理端）和 `http://127.0.0.1:5174`（用户端）。
3. 把 Client ID 填到 **Google Client ID**，Client Secret 可选、仅保存在服务端，然后打开 **Google 登录**。
4. 打开 **Google 注册** 后，首次 Google 登录会建号。用户端注册获得 `member` 角色；管理端自助注册不额外赋权。已有邮箱会在账号类型与客户端一致时绑定 Google。

#### 人机验证

`auth.captcha_provider`：

| 值           | 行为 |
| ------------ | ---- |
| `none`       | 不验证 |
| `image`      | 内置图形验证码（默认；`GET /api/v1/auth/captcha`） |
| `recaptcha`  | reCAPTCHA v3；分数低于 `auth.recaptcha_min_score`（默认 `0.5`）且已配置 v2 时，前端回退勾选框 |
| `turnstile`  | Cloudflare Turnstile |

站点密钥可公开。服务端密钥只存在服务端，配置列表中会掩码。未设置 `auth.captcha_provider` 时仍回退 `app.captcha_enabled`。

本地测 Turnstile 用 [Cloudflare dummy key](https://developers.cloudflare.com/turnstile/troubleshooting/testing/)（人机验证站点密钥，不是 API Token），`localhost` 可用。Dummy sitekey 会签发 `XXXX.DUMMY.TOKEN.XXXX`。通过用的 dummy secret 始终校验成功，失败用的始终拒绝。

| Site key | Secret | 结果 |
| --- | --- | --- |
| `1x00000000000000000000AA`（可见、始终通过） | `1x0000000000000000000000000000000AA` | 登录成功 |
| `2x00000000000000000000AB`（可见、始终失败） | `2x0000000000000000000000000000000AA` | 组件 / 校验失败 |
| `1x00000000000000000000BB`（不可见、始终通过） | `1x0000000000000000000000000000000AA` | 登录成功 |
| `3x00000000000000000000FF`（强制交互） | `1x0000000000000000000000000000000AA` | 浏览器里弹出挑战 |

`3x0000000000000000000000000000000AA` 返回 `timeout-or-duplicate`（token 已用）。生产 secret 会拒绝 dummy token。

### 测试

```bash
cd server && go test ./...
```

### Docker

```bash
JWT_SECRET='replace-with-32+chars' MAIL_UNSUB_SECRET='another-32+chars' BOOTSTRAP_ADMIN_PASSWORD='YourAdmin1' docker compose up --build -d
```

管理端 `:5173`，接口 `:8080`。`GET /health` 返回包含 `ok` 的 JSON。

### 运维

- 表结构变更放在 `server/internal/migrate/sql`，服务启动时自动执行。
- 登录与写操作会写入 `op_logs`，管理端 `/logs` 可查。
- `GET /metrics` 提供 Prometheus 风格请求计数。
