package httpserver

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/captcha"
	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/googleid"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/rbac"
	"go-react-shadcn/internal/security"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/store"
	"go-react-shadcn/internal/token"
	"gorm.io/gorm"
)

type App struct {
	Cfg              config.Config
	DB               *gorm.DB
	Captcha          *captcha.Service
	Tokens           *token.Service
	Enforcer         *casbin.Enforcer
	LoginGuard       *security.LoginGuard
	ForgotGuard      *security.LoginGuard
	ResetGuard       *security.LoginGuard
	ResetTokenGuard  *security.LoginGuard
	UnsubGuard       *security.LoginGuard
	VerifyGuard      *security.LoginGuard
	VerifyTokenGuard *security.LoginGuard
	Mail             mailer.Sender
	HTTP             *http.Client
	GoogleVerify     googleid.Verifier
	SiteVerify       siteverifyClient
	MailQ            *mailer.Queue
	Router           *gin.Engine
	metrics          *httpMetrics
	sessions         *sessionCache
	syscfg           *sysCache
	depts            *deptCache
	apiLogs          *apiLogQueue
	mailDB           *gorm.DB
	importMu         sync.Mutex
	stopOnce         sync.Once
	stopCh           chan struct{}
	totpMu           sync.Mutex
	totpTickets      map[string]totpTicket
}

func New(cfg config.Config, db *gorm.DB) (*App, error) {
	enforcer, err := rbac.NewEnforcer(db)
	if err != nil {
		return nil, err
	}
	if cfg.UploadDir == "" {
		dir := filepath.Dir(cfg.DatabasePath)
		if dir == "." || dir == "" {
			dir = "data"
		}
		cfg.UploadDir = filepath.Join(dir, "uploads")
	}
	if err := seed.Run(db, enforcer, cfg.UploadDir, cfg.DevMode); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	if err := store.EnsureUserFTS(db); err != nil {
		return nil, fmt.Errorf("user fts: %w", err)
	}
	mailSender := &mailer.SMTP{DB: db, Key: cfg.JWTSecret}
	httpClient := &http.Client{Timeout: 8 * time.Second}
	mailQ := mailer.NewQueue(db, mailSender, cfg.UnsubSecret())
	mailQ.ConfigKey = cfg.JWTSecret
	app := &App{
		Cfg:              cfg,
		DB:               db,
		Captcha:          captcha.New(db, cfg.CaptchaDebug && cfg.DevMode),
		Tokens:           token.New(cfg.JWTSecret, cfg.JWTTTL),
		Enforcer:         enforcer,
		LoginGuard:       security.NewLoginGuard(),
		ForgotGuard:      security.NewIPLimiter(8, time.Minute),
		ResetGuard:       security.NewIPLimiter(12, time.Minute),
		ResetTokenGuard:  security.NewIPLimiter(5, time.Minute),
		UnsubGuard:       security.NewIPLimiter(20, time.Minute),
		VerifyGuard:      security.NewIPLimiter(12, time.Minute),
		VerifyTokenGuard: security.NewIPLimiter(5, time.Minute),
		Mail:             mailSender,
		HTTP:             httpClient,
		MailQ:            mailQ,
		metrics:          newHTTPMetrics(),
		sessions:         newSessionCache(cfg.SessionCache),
		syscfg:           newSysCache(30 * time.Second),
		depts:            &deptCache{},
		apiLogs:          newAPILogQueue(db, cfg.APILogEnabled, cfg.APILogSample),
		mailDB:           db,
		stopCh:           make(chan struct{}),
		totpTickets:      map[string]totpTicket{},
	}
	app.Router = app.buildRouter()
	app.warnSealedConfigs()
	go app.backgroundJobs()
	go app.MailQ.Run()
	return app, nil
}

func (a *App) backgroundJobs() {
	loginTick := time.NewTicker(time.Minute)
	purgeTick := time.NewTicker(time.Hour)
	defer loginTick.Stop()
	defer purgeTick.Stop()
	for {
		select {
		case <-loginTick.C:
			for _, g := range []*security.LoginGuard{a.LoginGuard, a.ForgotGuard, a.ResetGuard, a.ResetTokenGuard, a.UnsubGuard, a.VerifyGuard, a.VerifyTokenGuard} {
				if g != nil {
					g.Sweep()
				}
			}
			if a.sessions != nil {
				a.sessions.Sweep()
			}
			a.sweepTotpTickets(time.Now())
		case <-purgeTick.C:
			now := time.Now()
			if a.Cfg.LogRetentionDays > 0 {
				_ = a.purgeOldLogs(a.Cfg.LogRetentionDays)
			}
			_ = a.DB.Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", now, now.Add(-7*24*time.Hour)).
				Delete(&models.AuthSession{}).Error
		case <-a.stopCh:
			return
		}
	}
}

func (a *App) Close() {
	a.stopOnce.Do(func() {
		if a.stopCh != nil {
			close(a.stopCh)
		}
	})
	if a.MailQ != nil {
		a.MailQ.Stop()
	}
	if a.apiLogs != nil {
		a.apiLogs.Stop()
	}
	if a.mailDB != nil && a.mailDB != a.DB {
		_ = store.Close(a.mailDB)
	}
}

func (a *App) buildRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.HandleMethodNotAllowed = true
	a.configureTrustedProxies(r)
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		fail(c, http.StatusInternalServerError, CodeInternal, "internal error")
	}))
	r.Use(a.securityHeaders())
	r.Use(a.limitRequestBody())
	r.Use(gzipIfAsked())
	r.Use(a.traceMiddleware())
	r.Use(a.logAPIRequests())
	r.Use(a.observeRequests())
	r.Use(cors.New(cors.Config{
		AllowOriginFunc:  corsOriginAllowed(a.Cfg),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Trace-Id", "X-Metrics-Token", "X-Locale", "Accept-Language"},
		ExposeHeaders:    []string{"X-Trace-Id"},
		AllowCredentials: true,
	}))
	uploads := r.Group("/uploads")
	uploads.Use(a.uploadHeaders())
	uploads.Static("/", a.Cfg.UploadDir)
	r.GET("/health", a.handleHealth)
	r.GET("/live", a.handleLive)
	r.GET("/ready", a.handleReady)
	r.GET("/metrics", a.protectMetrics(), a.handleMetrics)
	r.GET("/openapi.yaml", a.handleOpenAPI)

	api := r.Group("/api/v1")
	api.GET("/health", a.handleHealth)
	api.GET("/openapi.yaml", a.handleOpenAPI)
	api.GET("/auth/captcha", a.handleCaptcha)
	api.GET("/auth/settings", a.handleAuthSettings)
	api.POST("/auth/login", a.handleLogin)
	api.POST("/auth/register", a.handleRegister)
	api.POST("/auth/verify-email", a.handleVerifyEmail)
	api.POST("/auth/google", a.handleGoogleAuth)
	api.POST("/auth/forgot-password", a.handleForgotPassword)
	api.POST("/auth/reset-password", a.handleResetPassword)
	api.POST("/mail/unsubscribe", a.handleUnsubscribe)

	totp := api.Group("/auth/totp")
	totp.Use(a.optionalJWT())
	totp.POST("/setup", a.handleTotpSetup)
	totp.POST("/confirm", a.handleTotpConfirm)
	totp.POST("/verify", a.handleTotpVerify)

	self := api.Group("")
	self.Use(a.requireJWT(), a.logMutations())
	self.GET("/auth/me", a.handleMe)
	self.GET("/auth/menus", a.handleMenus)
	self.GET("/auth/web-menus", a.handleWebMenus)
	self.PUT("/auth/profile", a.handleUpdateProfile)
	self.PUT("/auth/password", a.handleChangePassword)
	self.POST("/auth/logout", a.handleLogout)
	self.POST("/auth/avatar", a.handleUploadOwnAvatar)
	self.POST("/auth/totp/disable", a.handleTotpDisable)
	self.POST("/auth/totp/recovery", a.handleTotpRecovery)
	self.GET("/dicts/by/:code", a.handleLookupDict)
	self.GET("/notifications", a.handleListNotifications)
	self.GET("/notifications/unread-count", a.handleUnreadNotificationCount)
	self.POST("/notifications/read-all", a.handleReadAllNotifications)
	self.POST("/notifications/:id/read", a.handleReadNotification)

	authed := api.Group("")
	authed.Use(a.requireJWT(), a.requireCasbin(), a.logMutations())
	authed.GET("/dashboard/stats", a.handleDashboard)

	authed.GET("/users", a.handleListUsers)
	authed.GET("/users/export", a.handleExportUsers)
	authed.POST("/users/import", a.handleImportUsers)
	authed.GET("/users/import-jobs/:id", a.handleGetUserImportJob)
	authed.GET("/users/:id", a.handleGetUser)
	authed.POST("/users", a.handleCreateUser)
	authed.PUT("/users/:id", a.handleUpdateUser)
	authed.POST("/users/:id/revoke", a.handleRevokeUser)
	authed.GET("/users/:id/sessions", a.handleListUserSessions)
	authed.DELETE("/users/:id/sessions/:sid", a.handleRevokeUserSession)
	authed.DELETE("/users/:id", a.handleDeleteUser)
	authed.PUT("/users/:id/roles", a.handleAssignUserRoles)
	authed.POST("/users/:id/avatar", a.handleUploadUserAvatar)

	authed.GET("/roles", a.handleListRoles)
	authed.POST("/roles", a.handleCreateRole)
	authed.GET("/roles/:id", a.handleGetRole)
	authed.PUT("/roles/:id", a.handleUpdateRole)
	authed.DELETE("/roles/:id", a.handleDeleteRole)
	authed.PUT("/roles/:id/permissions", a.handleAssignRolePermissions)

	authed.GET("/permissions", a.handleListPermissions)
	authed.POST("/permissions", a.handleCreatePermission)
	authed.PUT("/permissions/:id", a.handleUpdatePermission)
	authed.DELETE("/permissions/:id", a.handleDeletePermission)

	authed.GET("/departments", a.handleListDepartments)
	authed.POST("/departments", a.handleCreateDepartment)
	authed.PUT("/departments/:id", a.handleUpdateDepartment)
	authed.DELETE("/departments/:id", a.handleDeleteDepartment)

	authed.GET("/dicts", a.handleListDicts)
	authed.POST("/dicts", a.handleCreateDict)
	authed.PUT("/dicts/:id", a.handleUpdateDict)
	authed.DELETE("/dicts/:id", a.handleDeleteDict)
	authed.GET("/dicts/:id/items", a.handleListDictItems)
	authed.POST("/dicts/:id/items", a.handleCreateDictItem)
	authed.PUT("/dict-items/:id", a.handleUpdateDictItem)
	authed.DELETE("/dict-items/:id", a.handleDeleteDictItem)

	authed.GET("/configs", a.handleListConfigs)
	authed.POST("/configs", a.handleCreateConfig)
	authed.PUT("/configs/batch", a.handleBatchConfigs)
	authed.PUT("/configs/:id", a.handleUpdateConfig)
	authed.DELETE("/configs/:id", a.handleDeleteConfig)
	authed.POST("/mail/test", a.handleTestMail)
	authed.GET("/mail/jobs", a.handleListMailJobs)
	authed.POST("/mail/jobs/:id/retry", a.handleRetryMailJob)
	authed.POST("/mail/jobs/:id/cancel", a.handleCancelMailJob)
	authed.GET("/mail/campaigns", a.handleListMailCampaigns)
	authed.GET("/mail/campaigns/:id", a.handleGetMailCampaign)
	authed.POST("/mail/campaigns", a.handleCreateMailCampaign)
	authed.PUT("/mail/campaigns/:id", a.handleUpdateMailCampaign)
	authed.DELETE("/mail/campaigns/:id", a.handleDeleteMailCampaign)
	authed.POST("/mail/campaigns/:id/schedule", a.handleScheduleMailCampaign)

	authed.GET("/logs/export", a.handleExportLogs)
	authed.GET("/logs", a.handleListLogs)
	authed.GET("/logs/login", a.handleListLoginLogs)
	authed.GET("/logs/api", a.handleListAPILogs)
	authed.DELETE("/logs", a.handleClearLogs)
	authed.POST("/logs/purge", a.handlePurgeLogs)

	authed.GET("/nav-menus", a.handleListNavMenus)
	authed.POST("/nav-menus", a.handleCreateNavMenu)
	authed.PUT("/nav-menus/reorder", a.handleReorderNavMenus)
	authed.PUT("/nav-menus/:id", a.handleUpdateNavMenu)
	authed.DELETE("/nav-menus/:id", a.handleDeleteNavMenu)
	authed.POST("/announcements", a.handleCreateAnnouncement)

	r.NoRoute(notFoundJSON)
	r.NoMethod(methodNotAllowedJSON)
	return r
}

func roleCodes(roles []models.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.Code)
	}
	return out
}

func hasRole(roles []models.Role, code string) bool {
	for _, r := range roles {
		if r.Code == code {
			return true
		}
	}
	return false
}

func parseIDs(raw []uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(raw))
	for _, id := range raw {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeMethod(m string) string {
	if m == "*" {
		return "*"
	}
	return strings.ToUpper(strings.TrimSpace(m))
}

func corsAllowOrigins(cfg config.Config) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		item := strings.TrimRight(strings.TrimSpace(raw), "/")
		if item == "" {
			return
		}
		if _, ok := seen[item]; ok {
			return
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	for _, part := range strings.Split(cfg.CORSOrigin, ",") {
		add(part)
	}
	if cfg.DevMode {
		add("http://127.0.0.1:5173")
		add("http://localhost:5173")
		add("http://127.0.0.1:5174")
		add("http://localhost:5174")
	}
	return out
}

func corsOriginAllowed(cfg config.Config) func(string) bool {
	allowed := map[string]struct{}{}
	for _, origin := range corsAllowOrigins(cfg) {
		allowed[origin] = struct{}{}
	}
	return func(origin string) bool {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		_, ok := allowed[origin]
		return ok
	}
}
