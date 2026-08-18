package seed

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	RoleAdmin        = "admin"
	RoleViewer       = "viewer"
	RoleOperator     = "operator"
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
		{"用户菜单", "user:list", "/api/v1/users", "GET", KindMenu, "进入用户页"},
		{"用户详情", "user:detail", "/api/v1/users/:id", "GET", KindAPI, "读取单个用户"},
		{"新建用户", "user:create", "/api/v1/users", "POST", KindButton, "用户页-新建按钮"},
		{"更新用户", "user:update", "/api/v1/users/:id", "PUT", KindButton, "用户页-更新按钮"},
		{"删除用户", "user:delete", "/api/v1/users/:id", "DELETE", KindButton, "用户页-删除按钮"},
		{"分配用户角色", "user:roles", "/api/v1/users/:id/roles", "PUT", KindButton, "用户页-角色勾选"},
		{"上传用户头像", "user:avatar", "/api/v1/users/:id/avatar", "POST", KindButton, "用户页-头像上传"},
		{"角色菜单", "role:list", "/api/v1/roles", "GET", KindMenu, "进入角色页"},
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
		{"删除参数", "config:delete", "/api/v1/configs/:id", "DELETE", KindButton, ""},
		{"日志菜单", "log:list", "/api/v1/logs", "GET", KindMenu, "进入操作日志页"},
		{"清空日志", "log:clear", "/api/v1/logs", "DELETE", KindButton, ""},
	}
}

func Run(db *gorm.DB, enforcer *casbin.Enforcer, uploadDir string) error {
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	perms, err := ensureCatalog(db)
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

	adminPerms := perms
	viewerPerms := pick(byCode, "me:read", "dashboard:read")
	operatorPerms := pick(byCode,
		"me:read", "dashboard:read",
		"user:list", "user:create",
		"role:list",
		"perm:list",
		"dict:list", "dict:item:list",
		"config:list",
		"log:list",
	)

	if err := db.Model(&adminRole).Association("Permissions").Replace(adminPerms); err != nil {
		return err
	}
	if err := db.Model(&viewerRole).Association("Permissions").Replace(viewerPerms); err != nil {
		return err
	}
	if err := db.Model(&operatorRole).Association("Permissions").Replace(operatorPerms); err != nil {
		return err
	}

	adminAvatar, err := writeSeedAvatar(uploadDir, "admin", "管", "#111111")
	if err != nil {
		return err
	}
	viewerAvatar, err := writeSeedAvatar(uploadDir, "viewer", "李", "#444444")
	if err != nil {
		return err
	}
	operatorAvatar, err := writeSeedAvatar(uploadDir, "operator", "张", "#222222")
	if err != nil {
		return err
	}
	adminUser, err := ensureUser(db, AdminUsername, AdminPassword, seedProfile{
		Nickname: "系统管理员", Avatar: adminAvatar, Email: "admin@latch.local", Phone: "13800000001",
		Gender: "male", Department: "技术部", Title: "负责人", Remark: "种子管理员账号",
	})
	if err != nil {
		return err
	}
	viewerUser, err := ensureUser(db, ViewerUsername, ViewerPassword, seedProfile{
		Nickname: "李访客", Avatar: viewerAvatar, Email: "viewer@latch.local", Phone: "13800000003",
		Gender: "female", Department: "市场部", Title: "观察员", Remark: "只读演示账号",
	})
	if err != nil {
		return err
	}
	operatorUser, err := ensureUser(db, OperatorUsername, OperatorPassword, seedProfile{
		Nickname: "张操作", Avatar: operatorAvatar, Email: "operator@latch.local", Phone: "13800000002",
		Gender: "male", Department: "运营部", Title: "运营专员", Remark: "按钮级权限演示",
	})
	if err != nil {
		return err
	}
	if err := db.Model(&adminUser).Association("Roles").Replace([]models.Role{adminRole}); err != nil {
		return err
	}
	if err := db.Model(&viewerUser).Association("Roles").Replace([]models.Role{viewerRole}); err != nil {
		return err
	}
	if err := db.Model(&operatorUser).Association("Roles").Replace([]models.Role{operatorRole}); err != nil {
		return err
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
	if err := SyncUserRoles(enforcer, adminUser.Username, []models.Role{adminRole}); err != nil {
		return err
	}
	if err := SyncUserRoles(enforcer, viewerUser.Username, []models.Role{viewerRole}); err != nil {
		return err
	}
	if err := SyncUserRoles(enforcer, operatorUser.Username, []models.Role{operatorRole}); err != nil {
		return err
	}
	if err := ensureDicts(db); err != nil {
		return err
	}
	if err := ensureConfigs(db); err != nil {
		return err
	}
	return nil
}

func IsSeedUsername(name string) bool {
	return name == AdminUsername || name == ViewerUsername || name == OperatorUsername
}

func IsSeedRole(code string) bool {
	return code == RoleAdmin || code == RoleViewer || code == RoleOperator
}

func SyncRolePolicies(enforcer *casbin.Enforcer, roleCode string, perms []models.Permission) error {
	return syncRolePolicies(enforcer, roleCode, perms)
}

func SyncUserRoles(enforcer *casbin.Enforcer, username string, roles []models.Role) error {
	if _, err := enforcer.RemoveFilteredGroupingPolicy(0, username); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := enforcer.AddGroupingPolicy(username, role.Code); err != nil {
			return err
		}
	}
	return nil
}

func RemoveUser(enforcer *casbin.Enforcer, username string) error {
	_, err := enforcer.RemoveFilteredGroupingPolicy(0, username)
	return err
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
	var role models.Role
	err := db.Where("code = ?", code).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = models.Role{Name: name, Code: code, Description: desc}
		return role, db.Create(&role).Error
	}
	if err != nil {
		return role, err
	}
	role.Name = name
	role.Description = desc
	return role, db.Save(&role).Error
}

type seedProfile struct {
	Nickname, Avatar, Email, Phone, Gender, Department, Title, Remark string
}

func ensureUser(db *gorm.DB, username, password string, profile seedProfile) (models.User, error) {
	var user models.User
	err := db.Where("username = ?", username).First(&user).Error
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
		}
		return user, db.Create(&user).Error
	}
	if err != nil {
		return user, err
	}
	user.Nickname = profile.Nickname
	user.Email = profile.Email
	user.Phone = profile.Phone
	user.Gender = profile.Gender
	user.Department = profile.Department
	user.Title = profile.Title
	if user.Avatar == "" {
		user.Avatar = profile.Avatar
	}
	if user.Remark == "" {
		user.Remark = profile.Remark
	}
	return user, db.Save(&user).Error
}

func writeSeedAvatar(uploadDir, username, letter, bg string) (string, error) {
	dir := filepath.Join(uploadDir, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := username + ".svg"
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="32" fill="%s"/>
  <text x="32" y="40" text-anchor="middle" fill="#ffffff" font-size="24" font-family="system-ui,sans-serif">%s</text>
</svg>`, bg, letter)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(svg), 0o644); err != nil {
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
		{"sys_user_status", "用户状态", []item{{"启用", "active"}, {"停用", "disabled"}}},
		{"sys_gender", "性别", []item{{"男", "male"}, {"女", "female"}, {"其他", "other"}}},
		{"sys_yes_no", "是否", []item{{"是", "1"}, {"否", "0"}}},
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
		{Key: "app.name", Value: "Latch Admin", Name: "系统名称", Group: "app", Remark: "浏览器标题与侧栏品牌"},
		{Key: "app.captcha_enabled", Value: "1", Name: "登录验证码", Group: "app", Remark: "1 开启 / 0 关闭"},
		{Key: "app.default_locale", Value: "zh-CN", Name: "默认语言", Group: "app", Remark: "zh-CN / zh-TW / en"},
	}
	for _, row := range rows {
		var existing models.SysConfig
		err := db.Where("key = ?", row.Key).First(&existing).Error
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
	for _, p := range perms {
		if _, err := enforcer.AddPolicy(roleCode, p.Path, p.Method); err != nil {
			return err
		}
	}
	return nil
}
