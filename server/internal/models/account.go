package models

import (
	"strings"

	"gorm.io/gorm"
)

const (
	AdminUserTable      = "admin_user"
	WebUserTable        = "web_user"
	AdminUserRolesTable = "user_roles"
	WebUserRolesTable   = "web_user_roles"
)

func NormalizeUserKind(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), UserKindWeb) {
		return UserKindWeb
	}
	return UserKindAdmin
}

func AccountTable(kind string) string {
	if NormalizeUserKind(kind) == UserKindWeb {
		return WebUserTable
	}
	return AdminUserTable
}

func RoleJoinTable(kind string) string {
	if NormalizeUserKind(kind) == UserKindWeb {
		return WebUserRolesTable
	}
	return AdminUserRolesTable
}

func Accounts(db *gorm.DB, kind string) *gorm.DB {
	return db.Table(AccountTable(kind))
}

func ReplaceUserRoles(tx *gorm.DB, kind string, userID uint, roles []Role) error {
	join := RoleJoinTable(kind)
	if err := tx.Exec("DELETE FROM "+join+" WHERE user_id = ?", userID).Error; err != nil {
		return err
	}
	if len(roles) == 0 {
		return nil
	}
	rows := make([]map[string]any, 0, len(roles))
	for _, r := range roles {
		rows = append(rows, map[string]any{"user_id": userID, "role_id": r.ID})
	}
	return tx.Table(join).Create(&rows).Error
}

func AttachRoles(tx *gorm.DB, kind string, users ...*User) error {
	kind = NormalizeUserKind(kind)
	ids := make([]uint, 0, len(users))
	byID := make(map[uint]*User, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		u.Kind = kind
		u.Roles = nil
		ids = append(ids, u.ID)
		byID[u.ID] = u
	}
	if len(ids) == 0 {
		return nil
	}
	var links []struct {
		UserID uint
		RoleID uint
	}
	if err := tx.Table(RoleJoinTable(kind)).Where("user_id IN ?", ids).Find(&links).Error; err != nil {
		return err
	}
	roleIDs := make([]uint, 0, len(links))
	seen := map[uint]struct{}{}
	for _, link := range links {
		if _, ok := seen[link.RoleID]; ok {
			continue
		}
		seen[link.RoleID] = struct{}{}
		roleIDs = append(roleIDs, link.RoleID)
	}
	if len(roleIDs) == 0 {
		return nil
	}
	var roles []Role
	if err := tx.Preload("Permissions").Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return err
	}
	roleByID := make(map[uint]Role, len(roles))
	for _, r := range roles {
		roleByID[r.ID] = r
	}
	for _, link := range links {
		u := byID[link.UserID]
		r, ok := roleByID[link.RoleID]
		if u == nil || !ok {
			continue
		}
		u.Roles = append(u.Roles, r)
	}
	return nil
}

func LoadAccount(tx *gorm.DB, kind string, dest *User, conds ...any) error {
	kind = NormalizeUserKind(kind)
	if err := Accounts(tx, kind).First(dest, conds...).Error; err != nil {
		return err
	}
	return AttachRoles(tx, kind, dest)
}

func DeleteAccountRoles(tx *gorm.DB, kind string, userID uint) error {
	return tx.Exec("DELETE FROM "+RoleJoinTable(kind)+" WHERE user_id = ?", userID).Error
}
