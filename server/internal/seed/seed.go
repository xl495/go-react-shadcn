package seed

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"gorm.io/gorm"
)

const (
	AdminUsername    = "admin"
	AdminPassword    = "admin123"
	ViewerUsername   = "viewer"
	ViewerPassword   = "viewer123"
	OperatorUsername = "operator"
	OperatorPassword = "operator123"
	MemberUsername   = "webuser"
	MemberPassword   = "webuser123"
	RoleAdmin        = "admin"
	RoleViewer       = "viewer"
	RoleOperator     = "operator"
	RoleMember       = "member"
	DictUserStatus   = "sys_user_status"
	DictGender       = "sys_gender"
	DictDepartment   = "sys_department"
	DictYesNo        = "sys_yes_no"
	DictPermKind     = "sys_perm_kind"
)

const (
	KindMenu   = "menu"
	KindButton = "button"
	KindAPI    = "api"
)

type catalogPerm struct {
	Name, Code, Path, Method, Kind, Description string
}

func catalog() []catalogPerm {
	return []catalogPerm{
		{"全部管理接口", "admin:all", "/api/v1/*", "*", KindAPI, "管理员通配权限"},
		{"查看自己", "me:read", "/api/v1/auth/me", "GET", KindAPI, "读取当前登录用户"},
		{"仪表盘", "dashboard:read", "/api/v1/dashboard/stats", "GET", KindMenu, "读取仪表盘统计"},
		{"组织管理", "org:menu", "/admin/org", "GET", KindMenu, "组织目录"},
		{"系统管理", "system:menu", "/admin/system", "GET", KindMenu, "系统目录"},
		{"用户菜单", "user:list", "/api/v1/users", "GET", KindMenu, "进入用户页"},
		{"用户详情", "user:detail", "/api/v1/users/:id", "GET", KindAPI, "读取单个用户"},
		{"新建用户", "user:create", "/api/v1/users", "POST", KindButton, "用户页-新建按钮"},
		{"更新用户", "user:update", "/api/v1/users/:id", "PUT", KindButton, "用户页-更新按钮"},
		{"删除用户", "user:delete", "/api/v1/users/:id", "DELETE", KindButton, "用户页-删除按钮"},
		{"分配用户角色", "user:roles", "/api/v1/users/:id/roles", "PUT", KindButton, "用户页-角色勾选"},
		{"上传用户头像", "user:avatar", "/api/v1/users/:id/avatar", "POST", KindButton, "用户页-头像上传"},
		{"导出用户", "user:export", "/api/v1/users/export", "GET", KindAPI, "导出用户 CSV"},
		{"导入用户", "user:import", "/api/v1/users/import", "POST", KindAPI, "导入用户 CSV"},
		{"导入任务", "user:import:read", "/api/v1/users/import-jobs/:id", "GET", KindAPI, "查看用户导入任务"},
		{"强制下线", "user:revoke", "/api/v1/users/:id/revoke", "POST", KindButton, "用户详情-踢下线"},
		{"会话列表", "user:sessions", "/api/v1/users/:id/sessions", "GET", KindAPI, "查看登录会话"},
		{"踢掉会话", "user:session:revoke", "/api/v1/users/:id/sessions/:sid", "DELETE", KindButton, "踢掉单个会话"},
		{"角色菜单", "role:list", "/api/v1/roles", "GET", KindMenu, "进入角色页"},
		{"查看角色", "role:detail", "/api/v1/roles/:id", "GET", KindAPI, "读取单个角色"},
		{"新建角色", "role:create", "/api/v1/roles", "POST", KindButton, "角色页-新建按钮"},
		{"更新角色", "role:update", "/api/v1/roles/:id", "PUT", KindButton, "角色页-更新按钮"},
		{"删除角色", "role:delete", "/api/v1/roles/:id", "DELETE", KindButton, "角色页-删除按钮"},
		{"分配角色权限", "role:perms", "/api/v1/roles/:id/permissions", "PUT", KindButton, "角色页-权限勾选"},
		{"权限菜单", "perm:list", "/api/v1/permissions", "GET", KindMenu, "进入权限页"},
		{"新建权限", "perm:create", "/api/v1/permissions", "POST", KindButton, "权限页-新建按钮"},
		{"更新权限", "perm:update", "/api/v1/permissions/:id", "PUT", KindButton, "权限页-更新按钮"},
		{"删除权限", "perm:delete", "/api/v1/permissions/:id", "DELETE", KindButton, "权限页-删除按钮"},
		{"字典菜单", "dict:list", "/api/v1/dicts", "GET", KindMenu, "进入字典页"},
		{"新建字典", "dict:create", "/api/v1/dicts", "POST", KindButton, ""},
		{"更新字典", "dict:update", "/api/v1/dicts/:id", "PUT", KindButton, ""},
		{"删除字典", "dict:delete", "/api/v1/dicts/:id", "DELETE", KindButton, ""},
		{"字典项列表", "dict:item:list", "/api/v1/dicts/:id/items", "GET", KindAPI, ""},
		{"新建字典项", "dict:item:create", "/api/v1/dicts/:id/items", "POST", KindButton, ""},
		{"更新字典项", "dict:item:update", "/api/v1/dict-items/:id", "PUT", KindButton, ""},
		{"删除字典项", "dict:item:delete", "/api/v1/dict-items/:id", "DELETE", KindButton, ""},
		{"参数菜单", "config:list", "/api/v1/configs", "GET", KindMenu, "进入系统参数页"},
		{"新建参数", "config:create", "/api/v1/configs", "POST", KindButton, ""},
		{"更新参数", "config:update", "/api/v1/configs/:id", "PUT", KindButton, ""},
		{"批量保存参数", "config:batch", "/api/v1/configs/batch", "PUT", KindAPI, ""},
		{"删除参数", "config:delete", "/api/v1/configs/:id", "DELETE", KindButton, ""},
		{"发送测试邮件", "mail:test", "/api/v1/mail/test", "POST", KindButton, "系统参数页-测试发信"},
		{"邮件队列", "mail:jobs:list", "/api/v1/mail/jobs", "GET", KindMenu, "查看投递队列"},
		{"重试邮件", "mail:jobs:retry", "/api/v1/mail/jobs/:id/retry", "POST", KindButton, ""},
		{"取消邮件", "mail:jobs:cancel", "/api/v1/mail/jobs/:id/cancel", "POST", KindButton, ""},
		{"邮件模板", "mail:campaign:list", "/api/v1/mail/campaigns", "GET", KindMenu, "邮件模板"},
		{"查看模板", "mail:campaign:detail", "/api/v1/mail/campaigns/:id", "GET", KindAPI, ""},
		{"新建模板", "mail:campaign:create", "/api/v1/mail/campaigns", "POST", KindButton, ""},
		{"更新模板", "mail:campaign:update", "/api/v1/mail/campaigns/:id", "PUT", KindButton, ""},
		{"删除模板", "mail:campaign:delete", "/api/v1/mail/campaigns/:id", "DELETE", KindButton, ""},
		{"投放模板", "mail:campaign:schedule", "/api/v1/mail/campaigns/:id/schedule", "POST", KindButton, ""},
		{"日志菜单", "log:list", "/api/v1/logs", "GET", KindMenu, "进入操作日志页"},
		{"导出日志", "log:export", "/api/v1/logs/export", "GET", KindAPI, "导出操作日志 CSV"},
		{"登录日志", "log:login:list", "/api/v1/logs/login", "GET", KindAPI, "查询登录日志"},
		{"API日志", "log:api:list", "/api/v1/logs/api", "GET", KindAPI, "查询 API 日志"},
		{"清空日志", "log:clear", "/api/v1/logs", "DELETE", KindButton, ""},
		{"滚动清理", "log:purge", "/api/v1/logs/purge", "POST", KindButton, "按保留天数清理"},
		{"部门列表", "dept:list", "/api/v1/departments", "GET", KindMenu, "查询部门树"},
		{"新建部门", "dept:create", "/api/v1/departments", "POST", KindButton, ""},
		{"更新部门", "dept:update", "/api/v1/departments/:id", "PUT", KindButton, ""},
		{"删除部门", "dept:delete", "/api/v1/departments/:id", "DELETE", KindButton, ""},
		{"菜单树", "menu:read", "/api/v1/auth/menus", "GET", KindAPI, "读取当前用户菜单"},
		{"用户端菜单", "menu:web:read", "/api/v1/auth/web-menus", "GET", KindAPI, "读取用户端菜单"},
		{"菜单管理", "menu:list", "/api/v1/nav-menus", "GET", KindMenu, "维护导航树"},
		{"新建菜单", "menu:create", "/api/v1/nav-menus", "POST", KindButton, ""},
		{"更新菜单", "menu:update", "/api/v1/nav-menus/:id", "PUT", KindButton, ""},
		{"删除菜单", "menu:delete", "/api/v1/nav-menus/:id", "DELETE", KindButton, ""},
		{"站内通知", "notify:list", "/api/v1/notifications", "GET", KindMenu, "查看站内通知"},
		{"发布公告", "announce:create", "/api/v1/announcements", "POST", KindButton, "按端发布公告"},
	}
}

func CasbinSub(kind string, userID uint) string {
	return models.NormalizeUserKind(kind) + ":" + strconv.FormatUint(uint64(userID), 10)
}

func IsDefaultPassword(plain string) bool {
	return passwd.IsBuiltinSeedPassword(plain)
}

func Run(db *gorm.DB, enforcer *casbin.Enforcer, uploadDir string, autoMigrate bool) error {
	if _, err := ensureCatalog(db); err != nil {
		return err
	}
	if err := syncMenuMeta(db); err != nil {
		return err
	}
	if err := syncMenuParents(db); err != nil {
		return err
	}
	if err := ensureNavMenus(db); err != nil {
		return err
	}
	if err := ensureDepartments(db); err != nil {
		return err
	}
	perms, err := loadAllPerms(db)
	if err != nil {
		return err
	}
	byCode := map[string]models.Permission{}
	for _, p := range perms {
		byCode[p.Code] = p
	}

	adminRole, err := ensureRole(db, "管理员", RoleAdmin, "全部权限")
	if err != nil {
		return err
	}
	viewerRole, err := ensureRole(db, "访客", RoleViewer, "仅仪表盘与个人信息")
	if err != nil {
		return err
	}
	operatorRole, err := ensureRole(db, "操作员", RoleOperator, "能看管理页，但只有部分按钮")
	if err != nil {
		return err
	}
	memberRole, err := ensureRole(db, "会员", RoleMember, "用户端账号，仅个人资料")
	if err != nil {
		return err
	}

	adminPerms := perms
	viewerPerms := pick(byCode, "me:read", "dashboard:read", "notify:list")
	operatorPerms := pick(byCode,
		"me:read", "dashboard:read", "notify:list",
		"user:list", "user:create", "user:export", "user:import", "user:import:read",
		"role:list", "role:detail",
		"perm:list",
		"dict:list", "dict:item:list",
		"config:list",
		"log:list",
		"mail:jobs:list", "mail:campaign:list", "mail:campaign:detail",
	)
	memberPerms := pick(byCode, "me:read", "notify:list")

	if err := db.Model(&adminRole).Association("Permissions").Replace(adminPerms); err != nil {
		return err
	}
	if err := db.Model(&viewerRole).Association("Permissions").Replace(viewerPerms); err != nil {
		return err
	}
	if err := db.Model(&operatorRole).Association("Permissions").Replace(operatorPerms); err != nil {
		return err
	}
	if err := db.Model(&memberRole).Association("Permissions").Replace(memberPerms); err != nil {
		return err
	}

	adminAvatar, err := writeSeedAvatar(uploadDir, "admin", "管", "#111111")
	if err != nil {
		return err
	}
	adminPass := AdminPassword
	if !autoMigrate {
		if v := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")); v != "" {
			if err := passwd.CheckProduction(v, AdminUsername); err != nil {
				return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD: %w", err)
			}
			adminPass = v
		} else {
			var n int64
			_ = models.Accounts(db, models.UserKindAdmin).Where("username = ?", AdminUsername).Count(&n).Error
			if n == 0 {
				return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD is required in production to create the admin user")
			}
		}
	}
	adminUser, err := ensureUser(db, AdminUsername, adminPass, seedProfile{
		Nickname: "系统管理员", Avatar: adminAvatar, Email: "admin@latch.local", Phone: "13800000001",
		Gender: "male", Department: "tech", Title: "负责人", Remark: "种子管理员账号",
		Kind: models.UserKindAdmin,
	})
	if err != nil {
		return err
	}
	if err := models.ReplaceUserRoles(db, adminUser.Kind, adminUser.ID, []models.Role{adminRole}); err != nil {
		return err
	}
	if autoMigrate {
		viewerAvatar, err := writeSeedAvatar(uploadDir, "viewer", "李", "#444444")
		if err != nil {
			return err
		}
		operatorAvatar, err := writeSeedAvatar(uploadDir, "operator", "张", "#222222")
		if err != nil {
			return err
		}
		memberAvatar, err := writeSeedAvatar(uploadDir, "webuser", "王", "#2563eb")
		if err != nil {
			return err
		}
		viewerUser, err := ensureUser(db, ViewerUsername, ViewerPassword, seedProfile{
			Nickname: "李访客", Avatar: viewerAvatar, Email: "viewer@latch.local", Phone: "13800000003",
			Gender: "female", Department: "market", Title: "观察员", Remark: "只读演示账号",
			Kind: models.UserKindAdmin,
		})
		if err != nil {
			return err
		}
		operatorUser, err := ensureUser(db, OperatorUsername, OperatorPassword, seedProfile{
			Nickname: "张操作", Avatar: operatorAvatar, Email: "operator@latch.local", Phone: "13800000002",
			Gender: "male", Department: "ops", Title: "运营专员", Remark: "按钮级权限演示",
			Kind: models.UserKindAdmin,
		})
		if err != nil {
			return err
		}
		memberUser, err := ensureUser(db, MemberUsername, MemberPassword, seedProfile{
			Nickname: "王会员", Avatar: memberAvatar, Email: "webuser@latch.local", Phone: "13800000004",
			Gender: "male", Remark: "用户端演示账号",
			Kind: models.UserKindWeb,
		})
		if err != nil {
			return err
		}
		if err := models.ReplaceUserRoles(db, viewerUser.Kind, viewerUser.ID, []models.Role{viewerRole}); err != nil {
			return err
		}
		if err := models.ReplaceUserRoles(db, operatorUser.Kind, operatorUser.ID, []models.Role{operatorRole}); err != nil {
			return err
		}
		if err := models.ReplaceUserRoles(db, memberUser.Kind, memberUser.ID, []models.Role{memberRole}); err != nil {
			return err
		}
	}

	if err := syncRolePolicies(enforcer, adminRole.Code, adminPerms); err != nil {
		return err
	}
	if err := syncRolePolicies(enforcer, viewerRole.Code, viewerPerms); err != nil {
		return err
	}
	if err := syncRolePolicies(enforcer, operatorRole.Code, operatorPerms); err != nil {
		return err
	}
	if err := syncRolePolicies(enforcer, memberRole.Code, memberPerms); err != nil {
		return err
	}
	if err := syncAllUserGrouping(db, enforcer); err != nil {
		return err
	}
	if err := ensureDicts(db); err != nil {
		return err
	}
	if err := SyncDepartmentDict(db); err != nil {
		return err
	}
	if err := ensureConfigs(db); err != nil {
		return err
	}
	return syncUserDepartmentIDs(db)
}

func lookupDepartmentID(db *gorm.DB, code string) *uint {
	if code == "" {
		return nil
	}
	var dept models.Department
	if err := db.Where("code = ?", code).First(&dept).Error; err != nil {
		return nil
	}
	return &dept.ID
}

func syncUserDepartmentIDs(db *gorm.DB) error {
	for _, kind := range []string{models.UserKindAdmin, models.UserKindWeb} {
		var users []models.User
		if err := models.Accounts(db, kind).Find(&users).Error; err != nil {
			return err
		}
		for _, u := range users {
			if u.Department == "" {
				continue
			}
			id := lookupDepartmentID(db, u.Department)
			if id == nil {
				continue
			}
			if u.DepartmentID != nil && *u.DepartmentID == *id {
				continue
			}
			if err := models.Accounts(db, kind).Where("id = ?", u.ID).Update("department_id", *id).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func IsSeedUsername(name string) bool {
	return name == AdminUsername || name == ViewerUsername || name == OperatorUsername || name == MemberUsername
}

func IsSeedRole(code string) bool {
	return code == RoleAdmin || code == RoleViewer || code == RoleOperator || code == RoleMember
}

func SyncRolePolicies(enforcer *casbin.Enforcer, roleCode string, perms []models.Permission) error {
	return syncRolePolicies(enforcer, roleCode, perms)
}

func SyncUserRoles(enforcer *casbin.Enforcer, subject string, roles []models.Role) error {
	if _, err := enforcer.RemoveFilteredGroupingPolicy(0, subject); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := enforcer.AddGroupingPolicy(subject, role.Code); err != nil {
			return err
		}
	}
	return nil
}

func RemoveUser(enforcer *casbin.Enforcer, subject string) error {
	_, err := enforcer.RemoveFilteredGroupingPolicy(0, subject)
	return err
}

func syncAllUserGrouping(db *gorm.DB, enforcer *casbin.Enforcer) error {
	for _, kind := range []string{models.UserKindAdmin, models.UserKindWeb} {
		var users []models.User
		if err := models.Accounts(db, kind).Find(&users).Error; err != nil {
			return err
		}
		ptrs := make([]*models.User, len(users))
		for i := range users {
			ptrs[i] = &users[i]
		}
		if err := models.AttachRoles(db, kind, ptrs...); err != nil {
			return err
		}
		for _, u := range users {
			_, _ = enforcer.RemoveFilteredGroupingPolicy(0, u.Username)
			if err := SyncUserRoles(enforcer, CasbinSub(u.Kind, u.ID), u.Roles); err != nil {
				return err
			}
		}
	}
	return nil
}

func RemoveRole(enforcer *casbin.Enforcer, roleCode string) error {
	if _, err := enforcer.RemoveFilteredPolicy(0, roleCode); err != nil {
		return err
	}
	_, err := enforcer.RemoveFilteredGroupingPolicy(1, roleCode)
	return err
}

func ensureCatalog(db *gorm.DB) ([]models.Permission, error) {
	out := make([]models.Permission, 0, len(catalog()))
	for _, item := range catalog() {
		var p models.Permission
		err := db.Where("code = ?", item.Code).First(&p).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p = models.Permission{
				Name:        item.Name,
				Code:        item.Code,
				Path:        item.Path,
				Method:      item.Method,
				Kind:        item.Kind,
				Description: item.Description,
			}
			if err := db.Create(&p).Error; err != nil {
				return nil, fmt.Errorf("create perm %s: %w", item.Code, err)
			}
		} else if err != nil {
			return nil, err
		} else {
			p.Name = item.Name
			p.Path = item.Path
			p.Method = item.Method
			p.Kind = item.Kind
			p.Description = item.Description
			if err := db.Save(&p).Error; err != nil {
				return nil, err
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func ensureRole(db *gorm.DB, name, code, desc string) (models.Role, error) {
	scope := models.DataScopeSelf
	switch code {
	case RoleAdmin:
		scope = models.DataScopeAll
	case RoleOperator:
		scope = models.DataScopeDept
	}
	var role models.Role
	err := db.Where("code = ?", code).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = models.Role{Name: name, Code: code, Description: desc, DataScope: scope}
		return role, db.Create(&role).Error
	}
	if err != nil {
		return role, err
	}
	if role.Name == "" {
		role.Name = name
	}
	if role.Description == "" {
		role.Description = desc
	}
	if role.DataScope == "" {
		role.DataScope = scope
	}
	return role, db.Save(&role).Error
}

type seedProfile struct {
	Nickname, Avatar, Email, Phone, Gender, Department, Title, Remark, Kind string
}

func ensureUser(db *gorm.DB, username, password string, profile seedProfile) (models.User, error) {
	kind := models.NormalizeUserKind(profile.Kind)
	var user models.User
	err := models.Accounts(db, kind).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, err := passwd.Hash(password)
		if err != nil {
			return user, err
		}
		user = models.User{
			Username: username, PasswordHash: hash, Status: "active",
			Nickname: profile.Nickname, Avatar: profile.Avatar, Email: profile.Email,
			Phone: profile.Phone, Gender: profile.Gender,
			Department: profile.Department, Title: profile.Title, Remark: profile.Remark,
			Timezone: mailerDefaultTZ(), MarketingOptIn: true, EmailVerified: true, Kind: kind,
		}
		if id := lookupDepartmentID(db, profile.Department); id != nil {
			user.DepartmentID = id
		}
		return user, models.Accounts(db, kind).Create(&user).Error
	}
	if err != nil {
		return user, err
	}
	user.Kind = kind
	user.Nickname = profile.Nickname
	user.Email = profile.Email
	user.Phone = profile.Phone
	user.Gender = profile.Gender
	user.Department = profile.Department
	user.Title = profile.Title
	user.EmailVerified = true
	if id := lookupDepartmentID(db, profile.Department); id != nil {
		user.DepartmentID = id
	}
	if user.Avatar == "" {
		user.Avatar = profile.Avatar
	}
	if user.Remark == "" {
		user.Remark = profile.Remark
	}
	if user.Timezone == "" {
		user.Timezone = mailerDefaultTZ()
	}
	return user, models.Accounts(db, kind).Omit("Roles", "Dept").Save(&user).Error
}

func mailerDefaultTZ() string { return "Asia/Shanghai" }

func writeSeedAvatar(uploadDir, username, letter, bg string) (string, error) {
	dir := filepath.Join(uploadDir, "avatars")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	name := username + ".svg"
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="32" fill="%s"/>
  <text x="32" y="40" text-anchor="middle" fill="#ffffff" font-size="24" font-family="system-ui,sans-serif">%s</text>
</svg>`, bg, letter)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(svg), 0o600); err != nil {
		return "", err
	}
	return "/uploads/avatars/" + name, nil
}

func ensureDicts(db *gorm.DB) error {
	type item struct{ Label, Value string }
	catalog := []struct {
		Code, Name string
		Items      []item
	}{
		{DictUserStatus, "用户状态", []item{{"启用", "active"}, {"停用", "disabled"}}},
		{DictGender, "性别", []item{{"男", "male"}, {"女", "female"}, {"其他", "other"}}},
		{DictDepartment, "部门", nil},
		{DictYesNo, "是否", []item{{"是", "1"}, {"否", "0"}}},
		{DictPermKind, "权限类型", []item{{"菜单", KindMenu}, {"按钮", KindButton}, {"接口", KindAPI}}},
	}
	for _, typ := range catalog {
		var dt models.DictType
		err := db.Where("code = ?", typ.Code).First(&dt).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dt = models.DictType{Code: typ.Code, Name: typ.Name, Status: "active"}
			if err := db.Create(&dt).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			dt.Name = typ.Name
			if err := db.Save(&dt).Error; err != nil {
				return err
			}
		}
		for i, it := range typ.Items {
			var row models.DictItem
			err := db.Where("type_code = ? AND value = ?", typ.Code, it.Value).First(&row).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row = models.DictItem{TypeCode: typ.Code, Label: it.Label, Value: it.Value, Sort: i, Status: "active"}
				if err := db.Create(&row).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureConfigs(db *gorm.DB) error {
	rows := []models.SysConfig{
		{Key: "app.name", Value: "gra", Name: "系统名称", Group: "app", Remark: "浏览器标题与侧栏品牌"},
		{Key: "app.captcha_enabled", Value: "1", Name: "登录验证码", Group: "app", Remark: "兼容项；优先使用 auth.captcha_provider"},
		{Key: "app.default_locale", Value: "zh-CN", Name: "默认语言", Group: "app", Remark: "zh-CN / en"},
		{Key: "auth.register_enabled", Value: "1", Name: "邮箱注册", Group: "auth", Remark: "1 允许用户端用户名密码注册"},
		{Key: "auth.admin_totp_required", Value: "0", Name: "管理端强制 TOTP", Group: "auth", Remark: "1 要求后台账号绑定二次验证"},
		{Key: "auth.google_enabled", Value: "0", Name: "Google 登录", Group: "auth", Remark: "1 开启 / 0 关闭"},
		{Key: "auth.google_register_enabled", Value: "0", Name: "Google 注册", Group: "auth", Remark: "1 允许用 Google 创建新账号"},
		{Key: "auth.google_client_id", Value: "", Name: "Google Client ID", Group: "auth", Remark: "OAuth / GIS 客户端 ID"},
		{Key: "auth.google_client_secret", Value: "", Name: "Google Client Secret", Group: "auth", Remark: "列表中不会回显明文"},
		{Key: "auth.captcha_provider", Value: "image", Name: "验证码提供方", Group: "auth", Remark: "none / image / recaptcha / turnstile"},
		{Key: "auth.recaptcha_site_key_v3", Value: "", Name: "reCAPTCHA v3 Site Key", Group: "auth", Remark: "前端 v3 站点密钥"},
		{Key: "auth.recaptcha_secret_v3", Value: "", Name: "reCAPTCHA v3 Secret", Group: "auth", Remark: "服务端 v3 密钥"},
		{Key: "auth.recaptcha_site_key_v2", Value: "", Name: "reCAPTCHA v2 Site Key", Group: "auth", Remark: "v3 失败时回退的勾选框密钥"},
		{Key: "auth.recaptcha_secret_v2", Value: "", Name: "reCAPTCHA v2 Secret", Group: "auth", Remark: "服务端 v2 密钥"},
		{Key: "auth.recaptcha_min_score", Value: "0.5", Name: "reCAPTCHA 最低分", Group: "auth", Remark: "v3 低于此分数则回退 v2"},
		{Key: "auth.turnstile_site_key", Value: "", Name: "Turnstile Site Key", Group: "auth", Remark: "前端站点密钥；本地可用 dummy 1x00000000000000000000AA"},
		{Key: "auth.turnstile_secret", Value: "", Name: "Turnstile Secret", Group: "auth", Remark: "服务端密钥；本地可用 dummy 1x0000000000000000000000000000000AA。不是 API Token"},
		{Key: "mail.enabled", Value: "0", Name: "启用发信", Group: "mail", Remark: "1 开启 / 0 关闭，关闭时不发送任何邮件"},
		{Key: "mail.host", Value: "", Name: "SMTP 主机", Group: "mail", Remark: "例如 smtp.example.com"},
		{Key: "mail.port", Value: "587", Name: "SMTP 端口", Group: "mail", Remark: "587 STARTTLS / 465 SSL"},
		{Key: "mail.username", Value: "", Name: "SMTP 用户名", Group: "mail", Remark: ""},
		{Key: "mail.password", Value: "", Name: "SMTP 密码", Group: "mail", Remark: "列表中不会回显明文"},
		{Key: "mail.from", Value: "", Name: "发件人邮箱", Group: "mail", Remark: "MAIL FROM 地址"},
		{Key: "mail.from_name", Value: "gra", Name: "发件人名称", Group: "mail", Remark: "收件箱显示的名称"},
		{Key: "mail.tls", Value: "starttls", Name: "加密方式", Group: "mail", Remark: "starttls / ssl / none"},
		{Key: "mail.reset_base_url", Value: "http://127.0.0.1:5173", Name: "重置链接前缀", Group: "mail", Remark: "忘记密码邮件中的站点地址，用户端可改为 :5174"},
		{Key: "mail.default_timezone", Value: "Asia/Shanghai", Name: "默认时区", Group: "mail", Remark: "IANA，用户未填时区时使用"},
		{Key: "mail.quiet_start", Value: "22:00", Name: "静默开始", Group: "mail", Remark: "本地时间，运营/营销在此之后不发送"},
		{Key: "mail.quiet_end", Value: "08:00", Name: "静默结束", Group: "mail", Remark: "可跨午夜"},
		{Key: "mail.marketing_start", Value: "09:00", Name: "营销开始", Group: "mail", Remark: "营销邮件允许发送的本地开始时间"},
		{Key: "mail.marketing_end", Value: "21:00", Name: "营销结束", Group: "mail", Remark: "营销邮件允许发送的本地结束时间"},
		{Key: "mail.rate_per_minute", Value: "30", Name: "每分钟封数", Group: "mail", Remark: "Worker 全局 SMTP 上限"},
		{Key: "mail.max_attempts", Value: "5", Name: "最大重试", Group: "mail", Remark: "超过后进入死信"},
		{Key: "mail.worker_tick_ms", Value: "2000", Name: "Worker 间隔", Group: "mail", Remark: "毫秒；紧急邮件会额外唤醒"},
	}
	for _, row := range rows {
		var existing models.SysConfig
		err := db.Where(`"key" = ?`, row.Key).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&row).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		existing.Name = row.Name
		existing.Group = row.Group
		existing.Remark = row.Remark
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	}
	if err := db.Model(&models.SysConfig{}).Where(`"key" = ? AND value IN ?`, "app.name", []string{"Latch Admin", "Latch"}).Update("value", "gra").Error; err != nil {
		return err
	}
	if err := db.Model(&models.SysConfig{}).Where(`"key" = ? AND value = ?`, "mail.from_name", "Latch").Update("value", "gra").Error; err != nil {
		return err
	}
	return nil
}

func pick(byCode map[string]models.Permission, codes ...string) []models.Permission {
	out := make([]models.Permission, 0, len(codes))
	for _, c := range codes {
		if p, ok := byCode[c]; ok {
			out = append(out, p)
		}
	}
	return out
}

func syncRolePolicies(enforcer *casbin.Enforcer, roleCode string, perms []models.Permission) error {
	if _, err := enforcer.RemoveFilteredPolicy(0, roleCode); err != nil {
		return err
	}
	hasUpdate := false
	hasBatch := false
	for _, p := range perms {
		if p.Code == "config:update" {
			hasUpdate = true
		}
		if p.Path == "/api/v1/configs/batch" && p.Method == "PUT" {
			hasBatch = true
		}
		if _, err := enforcer.AddPolicy(roleCode, p.Path, p.Method); err != nil {
			return err
		}
	}
	if hasUpdate && !hasBatch {
		if _, err := enforcer.AddPolicy(roleCode, "/api/v1/configs/batch", "PUT"); err != nil {
			return err
		}
	}
	return nil
}

func loadAllPerms(db *gorm.DB) ([]models.Permission, error) {
	var perms []models.Permission
	return perms, db.Order("id asc").Find(&perms).Error
}

func syncMenuMeta(db *gorm.DB) error {
	type row struct {
		route, icon, component string
		sort                   int
	}
	meta := map[string]row{
		"dashboard:read":     {"/", "LayoutDashboard", "DashboardPage", 10},
		"org:menu":           {"", "FolderTree", "", 15},
		"system:menu":        {"", "Monitor", "", 45},
		"user:list":          {"/users", "Users", "UsersPage", 20},
		"dept:list":          {"/departments", "Building2", "DepartmentsPage", 25},
		"role:list":          {"/roles", "Shield", "RolesPage", 30},
		"perm:list":          {"/permissions", "KeyRound", "PermissionsPage", 40},
		"dict:list":          {"/dicts", "BookMarked", "DictsPage", 50},
		"config:list":        {"/configs", "Settings2", "ConfigsPage", 60},
		"mail:jobs:list":     {"/mail/jobs", "Mail", "MailJobsPage", 65},
		"mail:campaign:list": {"/mail/campaigns", "FileText", "MailCampaignsPage", 66},
		"log:list":           {"/logs", "ClipboardList", "LogsPage", 70},
		"menu:list":          {"/menus", "PanelTop", "MenusPage", 55},
		"notify:list":        {"/notifications", "Bell", "NotificationsPage", 12},
	}
	for code, m := range meta {
		if err := db.Model(&models.Permission{}).Where("code = ?", code).Updates(map[string]any{
			"route_path": m.route,
			"icon":       m.icon,
			"component":  m.component,
			"sort":       m.sort,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func syncMenuParents(db *gorm.DB) error {
	idByCode := func(code string) (uint, error) {
		var p models.Permission
		if err := db.Select("id").Where("code = ?", code).First(&p).Error; err != nil {
			return 0, err
		}
		return p.ID, nil
	}
	groups := map[string][]string{
		"org:menu": {
			"user:list", "dept:list", "role:list", "perm:list",
		},
		"system:menu": {
			"dict:list", "config:list", "menu:list", "mail:jobs:list", "mail:campaign:list", "log:list",
		},
	}
	for parentCode, children := range groups {
		pid, err := idByCode(parentCode)
		if err != nil {
			return fmt.Errorf("menu parent %s: %w", parentCode, err)
		}
		if err := db.Model(&models.Permission{}).Where("code IN ?", children).Update("parent_id", pid).Error; err != nil {
			return fmt.Errorf("menu children of %s: %w", parentCode, err)
		}
	}
	return nil
}

func SyncDepartmentDict(db *gorm.DB) error {
	var typ models.DictType
	err := db.Where("code = ?", DictDepartment).First(&typ).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		typ = models.DictType{Code: DictDepartment, Name: "部门", Status: "active"}
		if err := db.Create(&typ).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	var depts []models.Department
	if err := db.Order("sort asc, id asc").Find(&depts).Error; err != nil {
		return err
	}
	codes := make([]string, 0, len(depts))
	for i, d := range depts {
		codes = append(codes, d.Code)
		status := d.Status
		if status == "" {
			status = "active"
		}
		var row models.DictItem
		err := db.Where("type_code = ? AND value = ?", DictDepartment, d.Code).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = models.DictItem{
				TypeCode: DictDepartment, Label: d.Name, Value: d.Code, Sort: i, Status: status,
			}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		row.Label = d.Name
		row.Sort = i
		row.Status = status
		if err := db.Save(&row).Error; err != nil {
			return err
		}
	}
	q := db.Where("type_code = ?", DictDepartment)
	if len(codes) == 0 {
		return q.Delete(&models.DictItem{}).Error
	}
	return q.Where("value NOT IN ?", codes).Delete(&models.DictItem{}).Error
}

func ensureDepartments(db *gorm.DB) error {
	type seedDept struct {
		name, code string
		sort       int
	}
	for _, d := range []seedDept{
		{"总部", "hq", 1},
		{"技术部", "tech", 2},
		{"运营部", "ops", 3},
		{"市场部", "market", 4},
	} {
		var row models.Department
		err := db.Where("code = ?", d.code).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = models.Department{Name: d.name, Code: d.code, Sort: d.sort, Status: "active"}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		row.Name = d.name
		row.Sort = d.sort
		if err := db.Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
