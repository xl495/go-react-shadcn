package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/seed"
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
	LastLoginAt     any       `json:"lastLoginAt"`
	LastLoginIP     string    `json:"lastLoginIp"`
	Roles           []roleDTO `json:"roles"`
	PermissionCodes []string  `json:"permissionCodes"`
	CreatedAt       any       `json:"createdAt"`
	UpdatedAt       any       `json:"updatedAt"`
}

type roleDTO struct {
	ID          uint            `json:"id"`
	Name        string          `json:"name"`
	Code        string          `json:"code"`
	Description string          `json:"description"`
	Permissions []permissionDTO `json:"permissions,omitempty"`
}

type permissionDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Path        string `json:"path"`
	Method      string `json:"method"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type userProfileFields struct {
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Gender     string `json:"gender"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Remark     string `json:"remark"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Status   string `json:"status"`
	RoleIDs  []uint `json:"roleIds"`
	userProfileFields
}

type updateUserRequest struct {
	Password   *string `json:"password"`
	Status     *string `json:"status"`
	Nickname   *string `json:"nickname"`
	Avatar     *string `json:"avatar"`
	Email      *string `json:"email"`
	Phone      *string `json:"phone"`
	Gender     *string `json:"gender"`
	Department *string `json:"department"`
	Title      *string `json:"title"`
	Remark     *string `json:"remark"`
}

type assignRolesRequest struct {
	RoleIDs []uint `json:"roleIds"`
}

func toUserDTO(u models.User) userDTO {
	roles := make([]roleDTO, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, toRoleDTO(r, true))
	}
	return userDTO{
		ID:              u.ID,
		Username:        u.Username,
		Nickname:        u.Nickname,
		Avatar:          u.Avatar,
		Email:           u.Email,
		Phone:           u.Phone,
		Gender:          u.Gender,
		Department:      u.Department,
		Title:           u.Title,
		Remark:          u.Remark,
		Status:          u.Status,
		LastLoginAt:     u.LastLoginAt,
		LastLoginIP:     u.LastLoginIP,
		Roles:           roles,
		PermissionCodes: collectCodes(u),
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
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
	dto := roleDTO{ID: r.ID, Name: r.Name, Code: r.Code, Description: r.Description}
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
		ID:          p.ID,
		Name:        p.Name,
		Code:        p.Code,
		Path:        p.Path,
		Method:      p.Method,
		Kind:        p.Kind,
		Description: p.Description,
	}
}

func (a *App) handleGetUser(c *gin.Context) {
	var user models.User
	if err := a.DB.Preload("Roles.Permissions").First(&user, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "user not found")
		return
	}
	ok(c, toUserDTO(user))
}

func (a *App) handleListUsers(c *gin.Context) {
	var users []models.User
	if err := a.DB.Preload("Roles").Order("id asc").Find(&users).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50010, "failed to list users")
		return
	}
	out := make([]userDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toUserDTO(u))
	}
	ok(c, out)
}

func (a *App) handleCreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		fail(c, http.StatusBadRequest, 40010, "username and password required")
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	hash, err := passwd.Hash(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50011, "failed to hash password")
		return
	}
	user := models.User{
		Username: req.Username, PasswordHash: hash, Status: status,
		Nickname: req.Nickname, Avatar: req.Avatar, Email: req.Email, Phone: req.Phone,
		Gender: req.Gender, Department: req.Department, Title: req.Title, Remark: req.Remark,
	}
	if err := a.DB.Create(&user).Error; err != nil {
		fail(c, http.StatusConflict, 40910, "username already exists")
		return
	}
	roles, err := a.loadRoles(req.RoleIDs)
	if err != nil {
		fail(c, http.StatusBadRequest, 40011, "invalid role ids")
		return
	}
	if err := a.DB.Model(&user).Association("Roles").Replace(roles); err != nil {
		fail(c, http.StatusInternalServerError, 50012, "failed to assign roles")
		return
	}
	if err := seed.SyncUserRoles(a.Enforcer, user.Username, roles); err != nil {
		fail(c, http.StatusInternalServerError, 50013, "failed to sync rbac")
		return
	}
	user.Roles = roles
	ok(c, toUserDTO(user))
}

func (a *App) handleUpdateUser(c *gin.Context) {
	var user models.User
	if err := a.DB.Preload("Roles").First(&user, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "user not found")
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40012, "invalid request body")
		return
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := passwd.Hash(*req.Password)
		if err != nil {
			fail(c, http.StatusInternalServerError, 50011, "failed to hash password")
			return
		}
		user.PasswordHash = hash
	}
	if req.Status != nil && *req.Status != "" {
		user.Status = *req.Status
	}
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Gender != nil {
		user.Gender = *req.Gender
	}
	if req.Department != nil {
		user.Department = *req.Department
	}
	if req.Title != nil {
		user.Title = *req.Title
	}
	if req.Remark != nil {
		user.Remark = *req.Remark
	}
	if err := a.DB.Save(&user).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50014, "failed to update user")
		return
	}
	ok(c, toUserDTO(user))
}

func (a *App) handleDeleteUser(c *gin.Context) {
	var user models.User
	if err := a.DB.Preload("Roles").First(&user, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "user not found")
		return
	}
	if seed.IsSeedUsername(user.Username) {
		fail(c, http.StatusBadRequest, 40013, "cannot delete seeded user")
		return
	}
	if err := seed.RemoveUser(a.Enforcer, user.Username); err != nil {
		fail(c, http.StatusInternalServerError, 50015, "failed to sync rbac")
		return
	}
	if err := a.DB.Select("Roles").Delete(&user).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50016, "failed to delete user")
		return
	}
	ok(c, gin.H{"deleted": user.ID})
}

func (a *App) handleAssignUserRoles(c *gin.Context) {
	var user models.User
	if err := a.DB.First(&user, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "user not found")
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
	if err := a.DB.Model(&user).Association("Roles").Replace(roles); err != nil {
		fail(c, http.StatusInternalServerError, 50012, "failed to assign roles")
		return
	}
	if err := seed.SyncUserRoles(a.Enforcer, user.Username, roles); err != nil {
		fail(c, http.StatusInternalServerError, 50013, "failed to sync rbac")
		return
	}
	user.Roles = roles
	ok(c, toUserDTO(user))
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
