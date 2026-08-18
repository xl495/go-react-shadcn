package models

import "time"

type User struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Username     string     `json:"username" gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string     `json:"-" gorm:"not null"`
	Nickname     string     `json:"nickname" gorm:"size:64"`
	Avatar       string     `json:"avatar" gorm:"size:255"`
	Email        string     `json:"email" gorm:"size:128"`
	Phone        string     `json:"phone" gorm:"size:32"`
	Gender       string     `json:"gender" gorm:"size:16"`
	Department   string     `json:"department" gorm:"size:64"`
	Title        string     `json:"title" gorm:"size:64"`
	Remark       string     `json:"remark" gorm:"size:255"`
	Status       string     `json:"status" gorm:"size:16;not null;default:active"`
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	LastLoginIP  string     `json:"lastLoginIp" gorm:"size:64"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Roles        []Role     `json:"roles,omitempty" gorm:"many2many:user_roles;"`
}

type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"size:64;not null"`
	Code        string       `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Description string       `json:"description" gorm:"size:255"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	Users       []User       `json:"-" gorm:"many2many:user_roles;"`
}

type Permission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:64;not null"`
	Code        string    `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Path        string    `json:"path" gorm:"size:128;not null"`
	Method      string    `json:"method" gorm:"size:16;not null"`
	Kind        string    `json:"kind" gorm:"size:16;not null;default:api"`
	Description string    `json:"description" gorm:"size:255"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Roles       []Role    `json:"-" gorm:"many2many:role_permissions;"`
}

type DictType struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Code      string     `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Name      string     `json:"name" gorm:"size:64;not null"`
	Status    string     `json:"status" gorm:"size:16;not null;default:active"`
	Remark    string     `json:"remark" gorm:"size:255"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Items     []DictItem `json:"items,omitempty" gorm:"foreignKey:TypeCode;references:Code"`
}

type DictItem struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TypeCode  string    `json:"typeCode" gorm:"size:64;index;not null"`
	Label     string    `json:"label" gorm:"size:64;not null"`
	Value     string    `json:"value" gorm:"size:64;not null"`
	Sort      int       `json:"sort" gorm:"not null;default:0"`
	Status    string    `json:"status" gorm:"size:16;not null;default:active"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SysConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;size:64;not null"`
	Value     string    `json:"value" gorm:"size:512;not null"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	Group     string    `json:"group" gorm:"size:32;index"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type OpLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"size:64;index"`
	Module    string    `json:"module" gorm:"size:32"`
	Action    string    `json:"action" gorm:"size:32"`
	Method    string    `json:"method" gorm:"size:16"`
	Path      string    `json:"path" gorm:"size:128"`
	Status    int       `json:"status"`
	IP        string    `json:"ip" gorm:"size:64"`
	LatencyMs int64     `json:"latencyMs"`
	Detail    string    `json:"detail" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
}

func AllModels() []any {
	return []any{&User{}, &Role{}, &Permission{}, &DictType{}, &DictItem{}, &SysConfig{}, &OpLog{}}
}
