package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

type userDTO struct {
	ID              uint      `json:"id"`
	Username        string    `json:"username"`
	Nickname        string    `json:"nickname"`
	Avatar          string    `json:"avatar"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	Gender          string    `json:"gender"`
	Department      string    `json:"department"`
	Title           string    `json:"title"`
	Remark          string    `json:"remark"`
	Status          string    `json:"status"`
	Timezone        string    `json:"timezone"`
	MarketingOptIn  bool      `json:"marketingOptIn"`
	EmailVerified   bool      `json:"emailVerified"`
	Kind            string    `json:"kind"`
	TotpEnabled     bool      `json:"totpEnabled"`
	LastLoginAt     any       `json:"lastLoginAt"`
	LastLoginIP     string    `json:"lastLoginIp"`
	Roles           []roleDTO `json:"roles"`
	PermissionCodes []string  `json:"permissionCodes"`
	CreatedAt       any       `json:"createdAt"`
	UpdatedAt       any       `json:"updatedAt"`
}

type roleDTO struct {
	ID            uint            `json:"id"`
	Name          string          `json:"name"`
	Code          string          `json:"code"`
	Description   string          `json:"description"`
	DataScope     string          `json:"dataScope"`
	PermissionIDs []uint          `json:"permissionIds,omitempty"`
	Permissions   []permissionDTO `json:"permissions,omitempty"`
}

type permissionDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Path        string `json:"path"`
	Method      string `json:"method"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	ParentID    *uint  `json:"parentId,omitempty"`
	Sort        int    `json:"sort"`
	Icon        string `json:"icon"`
	RoutePath   string `json:"routePath"`
	Component   string `json:"component"`
	Hidden      bool   `json:"hidden"`
}

type userProfileFields struct {
	Nickname       string `json:"nickname"`
	Avatar         string `json:"avatar"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Gender         string `json:"gender"`
	Department     string `json:"department"`
	Title          string `json:"title"`
	Remark         string `json:"remark"`
	Timezone       string `json:"timezone"`
	MarketingOptIn *bool  `json:"marketingOptIn"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Status   string `json:"status"`
	Kind     string `json:"kind"`
	RoleIDs  []uint `json:"roleIds"`
	userProfileFields
}

type updateUserRequest struct {
	Password       *string `json:"password"`
	Status         *string `json:"status"`
	Nickname       *string `json:"nickname"`
	Avatar         *string `json:"avatar"`
	Email          *string `json:"email"`
	Phone          *string `json:"phone"`
	Gender         *string `json:"gender"`
	Department     *string `json:"department"`
	Title          *string `json:"title"`
	Remark         *string `json:"remark"`
	Timezone       *string `json:"timezone"`
	MarketingOptIn *bool   `json:"marketingOptIn"`
}

type assignRolesRequest struct {
	RoleIDs []uint `json:"roleIds"`
}

func (a *App) toUserDTO(u models.User) userDTO {
	return a.toUserDTOOpts(u, true)
}

func (a *App) toUserListDTO(u models.User) userDTO {
	return a.toUserDTOOpts(u, false)
}

func (a *App) toUserDTOOpts(u models.User, withPerms bool) userDTO {
	a.fillUserDepartments(&u)
	return toUserDTOOpts(u, withPerms)
}

func toUserDTOOpts(u models.User, withPerms bool) userDTO {
	roles := make([]roleDTO, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, toRoleDTO(r, withPerms))
	}
	dto := userDTO{
		ID:             u.ID,
		Username:       u.Username,
		Nickname:       u.Nickname,
		Avatar:         u.Avatar,
		Email:          u.Email,
		Phone:          u.Phone,
		Gender:         u.Gender,
		Department:     u.Department,
		Title:          u.Title,
		Remark:         u.Remark,
		Status:         u.Status,
		Timezone:       u.Timezone,
		MarketingOptIn: u.MarketingOptIn,
		EmailVerified:  u.EmailVerified,
		Kind:           normalizeUserKind(u.Kind),
		TotpEnabled:    u.TotpEnabled,
		LastLoginAt:    u.LastLoginAt,
		LastLoginIP:    u.LastLoginIP,
		Roles:          roles,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
	if withPerms {
		dto.PermissionCodes = collectCodes(u)
	}
	return dto
}

func collectCodes(u models.User) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, r := range u.Roles {
		if r.Code == seed.RoleAdmin {
			return []string{"*"}
		}
		for _, p := range r.Permissions {
			if p.Code == "" {
				continue
			}
			if _, ok := seen[p.Code]; ok {
				continue
			}
			seen[p.Code] = struct{}{}
			out = append(out, p.Code)
		}
	}
	return out
}

func toRoleDTO(r models.Role, withPerms bool) roleDTO {
	dto := roleDTO{ID: r.ID, Name: r.Name, Code: r.Code, Description: r.Description, DataScope: r.DataScope}
	if withPerms {
		dto.Permissions = make([]permissionDTO, 0, len(r.Permissions))
		for _, p := range r.Permissions {
			dto.Permissions = append(dto.Permissions, toPermissionDTO(p))
		}
	}
	return dto
}

func toPermissionDTO(p models.Permission) permissionDTO {
	return permissionDTO{
		ID: p.ID, Name: p.Name, Code: p.Code, Path: p.Path, Method: p.Method,
		Kind: p.Kind, Description: p.Description, ParentID: p.ParentID,
		Sort: p.Sort, Icon: p.Icon, RoutePath: p.RoutePath, Component: p.Component, Hidden: p.Hidden,
	}
}

func (a *App) handleGetUser(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"), "Roles.Permissions")
	if !found {
		return
	}
	ok(c, a.toUserDTO(user))
}

func (a *App) handleListUsers(c *gin.Context) {
	kind := strings.TrimSpace(c.Query("kind"))
	if kind != "" && kind != models.UserKindAdmin && kind != models.UserKindWeb {
		fail(c, http.StatusBadRequest, CodeInvalidUserBody, "invalid user kind")
		return
	}
	if kind == "" {
		kind = models.UserKindAdmin
	}
	actor, okActor := a.loadActor(c)
	if !okActor {
		return
	}
	p := parsePage(c, 20, 200)
	tbl := models.AccountTable(kind)
	q := a.applyUserDataScope(a.accounts(kind), actor, kind)
	q = applyEqual(q, tbl+".status", c.Query("status"))
	q = applyEqual(q, tbl+".gender", c.Query("gender"))
	q = a.applyDepartmentFilter(q, tbl, c.Query("department"))
	if roleID := parseQueryUint(c, "roleId"); roleID > 0 {
		q = q.Where(tbl+".id IN (SELECT user_id FROM "+models.RoleJoinTable(kind)+" WHERE role_id = ?)", roleID)
	}
	q = applyUserKeyword(q, tbl, c.Query("q"))
	total, okCount := countOrFail(c, q, CodeListUsers, "failed to list users")
	if !okCount {
		return
	}
	var users []models.User
	order := p.OrderClause(map[string]string{"id": "id", "username": "username"}, "id")
	if err := q.Order(order).Offset(p.Offset()).Limit(p.PageSize).Find(&users).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50010, "failed to list users")
		return
	}
	ptrs := make([]*models.User, len(users))
	for i := range users {
		ptrs[i] = &users[i]
	}
	if err := models.AttachRoles(a.DB, kind, ptrs...); err != nil {
		fail(c, http.StatusInternalServerError, 50010, "failed to list users")
		return
	}
	out := make([]userDTO, 0, len(users))
	for _, u := range users {
		out = append(out, a.toUserListDTO(u))
	}
	ok(c, pageResult[userDTO]{Items: out, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleCreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		fail(c, http.StatusBadRequest, 40010, "username and password required")
		return
	}
	if a.failIfWeakPassword(c, req.Password, req.Username) {
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	if !a.requireDictValue(c, seed.DictUserStatus, status) ||
		!a.requireDictValue(c, seed.DictGender, req.Gender) ||
		!a.requireDepartmentCode(c, req.Department) {
		return
	}
	if req.Kind != "" && req.Kind != models.UserKindAdmin && req.Kind != models.UserKindWeb {
		fail(c, http.StatusBadRequest, CodeInvalidUserBody, "invalid user kind")
		return
	}
	kind := normalizeUserKind(req.Kind)
	if req.Email != "" && a.emailTaken(kind, req.Email, 0) {
		fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
		return
	}
	hash, err := passwd.Hash(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50011, "failed to hash password")
		return
	}
	roles, err := a.loadRoles(req.RoleIDs)
	if err != nil {
		fail(c, http.StatusBadRequest, 40011, "invalid role ids")
		return
	}
	roles, err = a.defaultRolesForKind(kind, roles)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50012, "failed to create user")
		return
	}
	actor, loaded := a.loadActor(c)
	if !loaded {
		return
	}
	if a.rejectPrivilegedRoleGrant(c, actor, roles) {
		return
	}
	tz := mailer.DefaultTimezone
	if req.Timezone != "" {
		parsed, err := mailer.NormalizeTimezone(req.Timezone)
		if err != nil {
			fail(c, http.StatusBadRequest, CodeInvalidTimezone, "invalid timezone")
			return
		}
		tz = parsed
	}
	optIn := false
	if req.MarketingOptIn != nil {
		optIn = *req.MarketingOptIn
	}
	user := models.User{
		Username: req.Username, PasswordHash: hash, Status: status,
		Nickname: req.Nickname, Avatar: req.Avatar, Email: req.Email, Phone: req.Phone,
		Gender: req.Gender, Department: req.Department, Title: req.Title, Remark: req.Remark,
		Timezone: tz, MarketingOptIn: optIn, EmailVerified: true, Kind: kind,
	}
	a.applyDepartmentLink(&user)
	if err := a.withTx(func(tx *gorm.DB) error {
		if err := models.Accounts(tx, kind).Create(&user).Error; err != nil {
			return err
		}
		return models.ReplaceUserRoles(tx, kind, user.ID, roles)
	}); err != nil {
		if isUniqueViolation(err) {
			if req.Email != "" && a.emailTaken(kind, req.Email, 0) {
				fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
				return
			}
			fail(c, http.StatusConflict, 40910, "username already exists")
			return
		}
		fail(c, http.StatusInternalServerError, 50012, "failed to create user")
		return
	}
	if err := seed.SyncUserRoles(a.Enforcer, seed.CasbinSub(user.Kind, user.ID), roles); err != nil {
		fail(c, http.StatusInternalServerError, 50013, "failed to sync rbac")
		return
	}
	user.Roles = roles
	ok(c, a.toUserDTO(user))
}

func (a *App) handleUpdateUser(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"), "Roles")
	if !found {
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40012, "invalid request body")
		return
	}
	oldDTO := a.toUserDTO(user)
	revoke := false
	if req.Password != nil && *req.Password != "" {
		if a.failIfWeakPassword(c, *req.Password, user.Username) {
			return
		}
		hash, err := passwd.Hash(*req.Password)
		if err != nil {
			fail(c, http.StatusInternalServerError, 50011, "failed to hash password")
			return
		}
		user.PasswordHash = hash
		revoke = true
	}
	if req.Status != nil && *req.Status != "" {
		if !a.requireDictValue(c, seed.DictUserStatus, *req.Status) {
			return
		}
		user.Status = *req.Status
		if *req.Status != "active" {
			revoke = true
		}
	}
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.Avatar != nil {
		next := strings.TrimSpace(*req.Avatar)
		if next != "" && !validAvatarURL(next) {
			fail(c, http.StatusBadRequest, CodeAvatarType, "unsupported image type")
			return
		}
		user.Avatar = next
	}
	if req.Email != nil {
		if *req.Email != "" && a.emailTaken(user.Kind, *req.Email, user.ID) {
			fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
			return
		}
		user.Email = *req.Email
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Gender != nil {
		if !a.requireDictValue(c, seed.DictGender, *req.Gender) {
			return
		}
		user.Gender = *req.Gender
	}
	if req.Department != nil {
		if !a.requireDepartmentCode(c, *req.Department) {
			return
		}
		user.Department = *req.Department
		if strings.TrimSpace(*req.Department) == "" {
			user.DepartmentID = nil
		}
		a.applyDepartmentLink(&user)
	}
	if req.Title != nil {
		user.Title = *req.Title
	}
	if req.Remark != nil {
		user.Remark = *req.Remark
	}
	if req.Timezone != nil {
		tz, err := mailer.NormalizeTimezone(*req.Timezone)
		if err != nil {
			fail(c, http.StatusBadRequest, CodeInvalidTimezone, "invalid timezone")
			return
		}
		user.Timezone = tz
	}
	if req.MarketingOptIn != nil {
		user.MarketingOptIn = *req.MarketingOptIn
	}
	if revoke {
		user.TokenVersion++
	}
	if err := a.saveAccount(&user); err != nil {
		fail(c, http.StatusInternalServerError, 50014, "failed to update user")
		return
	}
	if revoke {
		a.revokeAuthSessions(user.Kind, user.ID)
		a.sessions.invalidate(user.Kind, user.ID)
	}
	a.recordOpLog(c, "user", "update", "update user "+user.Username, "", oldDTO, a.toUserDTO(user))
	ok(c, a.toUserDTO(user))
}

func (a *App) handleDeleteUser(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"), "Roles")
	if !found {
		return
	}
	if seed.IsSeedUsername(user.Username) {
		fail(c, http.StatusBadRequest, 40013, "cannot delete seeded user")
		return
	}
	a.sessions.invalidate(user.Kind, user.ID)
	removeUploadedFile(a.Cfg.UploadDir, user.Avatar)
	if err := a.withTx(func(tx *gorm.DB) error {
		if err := models.DeleteAccountRoles(tx, user.Kind, user.ID); err != nil {
			return err
		}
		return models.Accounts(tx, user.Kind).Delete(&user).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeDeleteUser, "failed to delete user")
		return
	}
	if err := seed.RemoveUser(a.Enforcer, seed.CasbinSub(user.Kind, user.ID)); err != nil {
		fail(c, http.StatusInternalServerError, CodeSyncRBAC, "failed to sync rbac")
		return
	}
	ok(c, gin.H{"deleted": user.ID})
}

func (a *App) handleRevokeUser(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"))
	if !found {
		return
	}
	if err := a.updateAccount(&user, map[string]any{"token_version": gorm.Expr("token_version + 1")}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return
	}
	a.revokeAuthSessions(user.Kind, user.ID)
	a.sessions.invalidate(user.Kind, user.ID)
	a.notify(user.Kind, user.ID, "revoke", "账号已强制下线", "", "user", user.ID)
	ok(c, gin.H{"revoked": user.ID})
}

func (a *App) handleListUserSessions(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"))
	if !found {
		return
	}
	var rows []models.AuthSession
	if err := a.DB.Where("user_kind = ? AND user_id = ?", user.Kind, user.ID).Order("id desc").Limit(50).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListUsers, "failed to list users")
		return
	}
	ok(c, rows)
}

func (a *App) handleRevokeUserSession(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"))
	if !found {
		return
	}
	var row models.AuthSession
	if err := a.DB.Where("id = ? AND user_kind = ? AND user_id = ?", c.Param("sid"), user.Kind, user.ID).First(&row).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "user not found")
		return
	}
	a.revokeAuthSessionJTI(row.JTI)
	ok(c, gin.H{"revoked": row.ID})
}

func (a *App) revokeAuthSessions(kind string, userID uint) {
	now := time.Now()
	_ = a.DB.Model(&models.AuthSession{}).
		Where("user_kind = ? AND user_id = ? AND revoked_at IS NULL", models.NormalizeUserKind(kind), userID).
		Update("revoked_at", now).Error
}

func (a *App) revokeAuthSessionJTI(jti string) {
	if jti == "" {
		return
	}
	now := time.Now()
	_ = a.DB.Model(&models.AuthSession{}).Where("jti = ? AND revoked_at IS NULL", jti).Update("revoked_at", now).Error
}

func (a *App) handleExportUsers(c *gin.Context) {
	kind := queryUserKind(c)
	actor, okActor := a.loadActor(c)
	if !okActor {
		return
	}
	q := a.applyUserDataScope(a.accounts(kind), actor, kind).Order("id asc")
	if err := streamCSV(c, a.DB, q, "users.csv",
		[]string{"id", "username", "nickname", "email", "phone", "status", "department", "kind"},
		func(u models.User) []string {
			a.fillUserDepartments(&u)
			return []string{formatUint(u.ID), u.Username, u.Nickname, u.Email, u.Phone, u.Status, u.Department, kind}
		},
	); err != nil && !c.Writer.Written() {
		fail(c, http.StatusInternalServerError, CodeListUsers, "failed to list users")
	}
}

func (a *App) handleAssignUserRoles(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"))
	if !found {
		return
	}
	var req assignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40014, "invalid request body")
		return
	}
	roles, err := a.loadRoles(req.RoleIDs)
	if err != nil {
		fail(c, http.StatusBadRequest, 40011, "invalid role ids")
		return
	}
	actor, loaded := a.loadActor(c)
	if !loaded {
		return
	}
	if a.rejectPrivilegedRoleGrant(c, actor, roles) {
		return
	}
	if err := a.withTx(func(tx *gorm.DB) error {
		return models.ReplaceUserRoles(tx, user.Kind, user.ID, roles)
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to assign roles")
		return
	}
	if err := seed.SyncUserRoles(a.Enforcer, seed.CasbinSub(user.Kind, user.ID), roles); err != nil {
		fail(c, http.StatusInternalServerError, 50013, "failed to sync rbac")
		return
	}
	user.Roles = roles
	a.notify(user.Kind, user.ID, "roles", "角色已变更", "", "user", user.ID)
	ok(c, a.toUserDTO(user))
}

func normalizeUserKind(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), models.UserKindWeb) {
		return models.UserKindWeb
	}
	return models.UserKindAdmin
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (a *App) emailTaken(kind, email string, exceptID uint) bool {
	email = normalizeEmail(email)
	if email == "" {
		return false
	}
	q := a.accounts(kind).Where("lower(email) = ? AND email <> ''", email)
	if exceptID > 0 {
		q = q.Where("id <> ?", exceptID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return true
	}
	return n > 0
}

func (a *App) rejectPrivilegedRoleGrant(c *gin.Context, actor models.User, roles []models.Role) bool {
	needAdmin := false
	for _, r := range roles {
		if r.Code == seed.RoleAdmin {
			needAdmin = true
			break
		}
	}
	if !needAdmin {
		return false
	}
	if a.userHasRoleCode(actor.Kind, actor.ID, seed.RoleAdmin) {
		return false
	}
	fail(c, http.StatusForbidden, CodePrivilegedRole, "cannot assign privileged role")
	return true
}

func (a *App) userHasRoleCode(kind string, userID uint, code string) bool {
	var n int64
	err := a.DB.Table(models.RoleJoinTable(kind)+" ur").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("ur.user_id = ? AND r.code = ?", userID, code).
		Count(&n).Error
	return err == nil && n > 0
}

func (a *App) defaultRolesForKind(kind string, roles []models.Role) ([]models.Role, error) {
	if kind != models.UserKindWeb || len(roles) > 0 {
		return roles, nil
	}
	var member models.Role
	if err := a.DB.Where("code = ?", seed.RoleMember).First(&member).Error; err != nil {
		return roles, nil
	}
	return []models.Role{member}, nil
}

func (a *App) loadRoles(ids []uint) ([]models.Role, error) {
	ids = parseIDs(ids)
	if len(ids) == 0 {
		return []models.Role{}, nil
	}
	var roles []models.Role
	if err := a.DB.Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	if len(roles) != len(ids) {
		return nil, gin.Error{Err: errRoleNotFound}
	}
	return roles, nil
}

var errRoleNotFound = errString("role not found")

type errString string

func (e errString) Error() string { return string(e) }
