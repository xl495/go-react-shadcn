package seed

import (
	"errors"

	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

type navSeed struct {
	Name, Code, PermCode, Route, Component, Icon, Parent string
	Sort                                                 int
}

func adminNavSeeds() []navSeed {
	return []navSeed{
		{"仪表盘", "dashboard:read", "dashboard:read", "/", "DashboardPage", "LayoutDashboard", "", 10},
		{"组织管理", "org:menu", "", "", "", "FolderTree", "", 15},
		{"后台用户", "user:list", "user:list", "/users", "UsersPage", "Users", "org:menu", 20},
		{"Web用户", "webuser:list", "user:list", "/web-users", "WebUsersPage", "Globe", "org:menu", 21},
		{"部门列表", "dept:list", "dept:list", "/departments", "DepartmentsPage", "Building2", "org:menu", 25},
		{"角色菜单", "role:list", "role:list", "/roles", "RolesPage", "Shield", "org:menu", 30},
		{"权限菜单", "perm:list", "perm:list", "/permissions", "PermissionsPage", "KeyRound", "org:menu", 40},
		{"系统管理", "system:menu", "", "", "", "Monitor", "", 45},
		{"字典菜单", "dict:list", "dict:list", "/dicts", "DictsPage", "BookMarked", "system:menu", 50},
		{"参数菜单", "config:list", "config:list", "/configs", "ConfigsPage", "Settings2", "system:menu", 60},
		{"邮件队列", "mail:jobs:list", "mail:jobs:list", "/mail/jobs", "MailJobsPage", "Mail", "system:menu", 65},
		{"邮件模板", "mail:campaign:list", "mail:campaign:list", "/mail/campaigns", "MailCampaignsPage", "FileText", "system:menu", 66},
		{"日志菜单", "log:list", "log:list", "/logs", "LogsPage", "ClipboardList", "system:menu", 70},
	}
}

func webNavSeeds() []navSeed {
	return []navSeed{
		{"首页", "web:home", "me:read", "/", "HomePage", "House", "", 10},
		{"我的资料", "web:profile", "me:read", "/profile", "ProfilePage", "User", "", 20},
		{"修改密码", "web:password", "me:read", "/password", "PasswordPage", "KeyRound", "", 30},
	}
}

func ensureNavMenus(db *gorm.DB) error {
	if err := upsertNavTable(db, "admin_menus", adminNavSeeds()); err != nil {
		return err
	}
	return upsertNavTable(db, "web_menus", webNavSeeds())
}

func upsertNavTable(db *gorm.DB, table string, seeds []navSeed) error {
	ids := map[string]uint{}
	for _, s := range seeds {
		id, err := upsertNavRow(db, table, models.NavMenu{
			Name:      s.Name,
			Code:      s.Code,
			PermCode:  s.PermCode,
			RoutePath: s.Route,
			Component: s.Component,
			Icon:      s.Icon,
			Sort:      s.Sort,
			Status:    "active",
		})
		if err != nil {
			return err
		}
		ids[s.Code] = id
	}
	for _, s := range seeds {
		var parentID *uint
		if s.Parent != "" {
			pid, ok := ids[s.Parent]
			if !ok {
				continue
			}
			parentID = &pid
		}
		if err := db.Table(table).Where("id = ?", ids[s.Code]).Update("parent_id", parentID).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertNavRow(db *gorm.DB, table string, row models.NavMenu) (uint, error) {
	var existing models.NavMenu
	err := db.Table(table).Where("code = ?", row.Code).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Table(table).Create(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	}
	if err != nil {
		return 0, err
	}
	err = db.Table(table).Where("id = ?", existing.ID).Updates(map[string]any{
		"name":       row.Name,
		"perm_code":  row.PermCode,
		"route_path": row.RoutePath,
		"component":  row.Component,
		"icon":       row.Icon,
		"sort":       row.Sort,
		"status":     row.Status,
		"hidden":     false,
	}).Error
	return existing.ID, err
}
