# Latch · React 19 + Gin + Casbin 登录权限系统

图形验证码登录、JWT 鉴权、Casbin RBAC、SQLite 持久化。管理端覆盖用户/角色/权限、数据字典、系统参数与操作日志。

## 目录

```
backend/     Go 1.24 + Gin + GORM/SQLite + Casbin + JWT + 图形验证码
frontend/    React 19 + Vite + Tailwind v4 + shadcn v4
```

## 种子账号

| 账号     | 密码         | 权限                                              |
| -------- | ------------ | ------------------------------------------------- |
| admin    | admin123     | 全部菜单与按钮                                    |
| operator | operator123  | 能进用户/角色/权限/字典/参数/日志页，但只有「新建用户」按钮 |
| viewer   | viewer123    | 仅仪表盘                                          |

权限分三类：`menu`（侧栏）、`button`（页面按钮）、`api`（纯接口）。前端按 `permissionCodes` 隐藏按钮，后端 Casbin 仍拦截对应 HTTP 方法。

界面支持 **简体中文 / 繁體中文 / English**。登录页右上角和侧栏可切换，选择会记在本地；接口错误按错误码翻译，种子角色/权限按编码翻译。

## 启动

前后端并发：

```bash
make dev
```

分别启动：

```bash
cd backend && go run ./cmd/server
cd frontend && npm run dev
```

调试登录时让验证码接口回传答案（仅本地验证用）：

```bash
CAPTCHA_DEBUG=1 go run ./cmd/server
```

生产构建：

```bash
cd frontend && npm run build
```

## 认证流程

1. `GET /api/v1/auth/captcha` 下发一次性图形验证码（`captchaId` + base64 图）。
2. `POST /api/v1/auth/login` 必须同时匹配用户名、密码与验证码，才签发 HS256 JWT。
3. 受保护接口带 `Authorization: Bearer <token>`。Casbin 用用户名 + Gin `FullPath` + Method 判定。
4. 角色权限写入 SQLite `casbin_rule`，`PUT /api/v1/roles/:id/permissions` 后立即改变 Enforce 结果。

## 测试

```bash
cd backend && go test ./...
```
