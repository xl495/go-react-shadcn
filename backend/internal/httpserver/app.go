package httpserver

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/captcha"
	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/rbac"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/token"
	"gorm.io/gorm"
)

type App struct {
	Cfg      config.Config
	DB       *gorm.DB
	Captcha  *captcha.Service
	Tokens   *token.Service
	Enforcer *casbin.Enforcer
	Router   *gin.Engine
}

func New(cfg config.Config, db *gorm.DB) (*App, error) {
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
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
	if err := seed.Run(db, enforcer, cfg.UploadDir); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	app := &App{
		Cfg:      cfg,
		DB:       db,
		Captcha:  captcha.New(cfg.CaptchaDebug),
		Tokens:   token.New(cfg.JWTSecret, cfg.JWTTTL),
		Enforcer: enforcer,
	}
	app.Router = app.buildRouter()
	return app, nil
}

func (a *App) buildRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{a.Cfg.CORSOrigin, "http://127.0.0.1:5173", "http://localhost:5173", "http://127.0.0.1:5174", "http://localhost:5174"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Static("/uploads", a.Cfg.UploadDir)

	api := r.Group("/api/v1")
	api.GET("/health", func(c *gin.Context) {
		ok(c, gin.H{"status": "up"})
	})
	api.GET("/auth/captcha", a.handleCaptcha)
	api.POST("/auth/login", a.handleLogin)

	self := api.Group("")
	self.Use(a.requireJWT(), a.logMutations())
	self.GET("/auth/me", a.handleMe)
	self.PUT("/auth/profile", a.handleUpdateProfile)
	self.PUT("/auth/password", a.handleChangePassword)
	self.POST("/auth/avatar", a.handleUploadOwnAvatar)
	self.GET("/dicts/by/:code", a.handleLookupDict)

	authed := api.Group("")
	authed.Use(a.requireJWT(), a.requireCasbin(), a.logMutations())
	authed.GET("/dashboard/stats", a.handleDashboard)

	authed.GET("/users", a.handleListUsers)
	authed.GET("/users/:id", a.handleGetUser)
	authed.POST("/users", a.handleCreateUser)
	authed.PUT("/users/:id", a.handleUpdateUser)
	authed.DELETE("/users/:id", a.handleDeleteUser)
	authed.PUT("/users/:id/roles", a.handleAssignUserRoles)
	authed.POST("/users/:id/avatar", a.handleUploadUserAvatar)

	authed.GET("/roles", a.handleListRoles)
	authed.POST("/roles", a.handleCreateRole)
	authed.PUT("/roles/:id", a.handleUpdateRole)
	authed.DELETE("/roles/:id", a.handleDeleteRole)
	authed.PUT("/roles/:id/permissions", a.handleAssignRolePermissions)

	authed.GET("/permissions", a.handleListPermissions)
	authed.POST("/permissions", a.handleCreatePermission)
	authed.PUT("/permissions/:id", a.handleUpdatePermission)
	authed.DELETE("/permissions/:id", a.handleDeletePermission)

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
	authed.PUT("/configs/:id", a.handleUpdateConfig)
	authed.DELETE("/configs/:id", a.handleDeleteConfig)

	authed.GET("/logs", a.handleListLogs)
	authed.DELETE("/logs", a.handleClearLogs)

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
