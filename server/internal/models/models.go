package models

import "time"

const (
	DataScopeAll        = "all"
	DataScopeDept       = "dept"
	DataScopeDeptAndSub = "dept_sub"
	DataScopeSelf       = "self"

	UserKindAdmin = "admin"
	UserKindWeb   = "web"
)

type Department struct {
	ID        uint         `json:"id" gorm:"primaryKey"`
	Name      string       `json:"name" gorm:"size:64;not null"`
	Code      string       `json:"code" gorm:"uniqueIndex;size:64;not null"`
	ParentID  *uint        `json:"parentId" gorm:"index"`
	Sort      int          `json:"sort" gorm:"not null;default:0"`
	Leader    string       `json:"leader" gorm:"size:64"`
	Status    string       `json:"status" gorm:"size:16;not null;default:active"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Children  []Department `json:"children,omitempty" gorm:"-"`
}

type User struct {
	ID               uint        `json:"id" gorm:"primaryKey"`
	Username         string      `json:"username" gorm:"uniqueIndex;size:64;not null"`
	PasswordHash     string      `json:"-" gorm:"not null"`
	Nickname         string      `json:"nickname" gorm:"size:64"`
	Avatar           string      `json:"avatar" gorm:"size:255"`
	Email            string      `json:"email" gorm:"size:128"`
	Phone            string      `json:"phone" gorm:"size:32"`
	Gender           string      `json:"gender" gorm:"size:16"`
	Department       string      `json:"department" gorm:"size:64"`
	DepartmentID     *uint       `json:"departmentId" gorm:"index"`
	Title            string      `json:"title" gorm:"size:64"`
	Remark           string      `json:"remark" gorm:"size:255"`
	Status           string      `json:"status" gorm:"size:16;not null;default:active"`
	TokenVersion     int         `json:"-" gorm:"not null;default:0"`
	FailedLoginCount int         `json:"-" gorm:"not null;default:0"`
	LockedUntil      *time.Time  `json:"-"`
	LastLoginAt      *time.Time  `json:"lastLoginAt"`
	LastLoginIP      string      `json:"lastLoginIp" gorm:"size:64"`
	Timezone         string      `json:"timezone" gorm:"size:64;not null;default:Asia/Shanghai"`
	MarketingOptIn   bool        `json:"marketingOptIn" gorm:"not null;default:false"`
	EmailVerified    bool        `json:"emailVerified" gorm:"not null;default:false"`
	Kind             string      `json:"kind" gorm:"-"`
	GoogleID         string      `json:"-" gorm:"size:64;index"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	Roles            []Role      `json:"roles,omitempty" gorm:"many2many:user_roles;"`
	Dept             *Department `json:"dept,omitempty" gorm:"foreignKey:DepartmentID"`
}

func (User) TableName() string { return AdminUserTable }

type webUserModel User

func (webUserModel) TableName() string { return WebUserTable }

type WebUserRole struct {
	UserID uint `gorm:"primaryKey;column:user_id"`
	RoleID uint `gorm:"primaryKey;column:role_id"`
}

func (WebUserRole) TableName() string { return WebUserRolesTable }

type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"size:64;not null"`
	Code        string       `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Description string       `json:"description" gorm:"size:255"`
	DataScope   string       `json:"dataScope" gorm:"size:16;not null;default:all"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	Users       []User       `json:"-" gorm:"many2many:user_roles;"`
}

type Permission struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"size:64;not null"`
	Code        string       `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Path        string       `json:"path" gorm:"size:128;not null"`
	Method      string       `json:"method" gorm:"size:16;not null"`
	Kind        string       `json:"kind" gorm:"size:16;not null;default:api"`
	Description string       `json:"description" gorm:"size:255"`
	ParentID    *uint        `json:"parentId" gorm:"index"`
	Sort        int          `json:"sort" gorm:"not null;default:0"`
	Icon        string       `json:"icon" gorm:"size:64"`
	RoutePath   string       `json:"routePath" gorm:"size:128"`
	Component   string       `json:"component" gorm:"size:128"`
	Hidden      bool         `json:"hidden" gorm:"not null;default:false"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Roles       []Role       `json:"-" gorm:"many2many:role_permissions;"`
	Children    []Permission `json:"children,omitempty" gorm:"-"`
}

type NavMenu struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ParentID  *uint     `json:"parentId" gorm:"index"`
	Audience  string    `json:"audience" gorm:"size:16;not null;uniqueIndex:idx_nav_audience_code"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	Code      string    `json:"code" gorm:"size:64;not null;uniqueIndex:idx_nav_audience_code"`
	RoutePath string    `json:"routePath" gorm:"size:128"`
	Component string    `json:"component" gorm:"size:128"`
	Icon      string    `json:"icon" gorm:"size:64"`
	Sort      int       `json:"sort" gorm:"not null;default:0"`
	Hidden    bool      `json:"hidden" gorm:"not null;default:false"`
	PermCode  string    `json:"permCode" gorm:"size:64;index"`
	Status    string    `json:"status" gorm:"size:16;not null;default:active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (NavMenu) TableName() string { return "nav_menu" }

const (
	NavAudienceAdmin   = "admin"
	NavAudienceWeb     = "web"
	TokenPurposeReset  = "reset"
	TokenPurposeVerify = "verify"
)

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

type LoginLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Username   string    `json:"username" gorm:"size:64;index;not null"`
	IP         string    `json:"ip" gorm:"size:64"`
	UserAgent  string    `json:"userAgent" gorm:"size:512"`
	Location   string    `json:"location" gorm:"size:128"`
	Status     string    `json:"status" gorm:"size:16;not null"`
	FailReason string    `json:"failReason" gorm:"size:255"`
	CreatedAt  time.Time `json:"createdAt"`
}

type OpLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TraceID     string    `json:"traceId" gorm:"size:64;index"`
	Username    string    `json:"username" gorm:"size:64;index"`
	Module      string    `json:"module" gorm:"size:32"`
	Action      string    `json:"action" gorm:"size:32"`
	Method      string    `json:"method" gorm:"size:16"`
	Path        string    `json:"path" gorm:"size:128"`
	Status      int       `json:"status"`
	IP          string    `json:"ip" gorm:"size:64"`
	LatencyMs   int64     `json:"latencyMs"`
	Detail      string    `json:"detail" gorm:"size:255"`
	Description string    `json:"description" gorm:"size:512"`
	OldValue    string    `json:"oldValue" gorm:"type:text"`
	NewValue    string    `json:"newValue" gorm:"type:text"`
	CreatedAt   time.Time `json:"createdAt"`
}

type APILog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	TraceID      string    `json:"traceId" gorm:"size:64;index;not null"`
	Username     string    `json:"username" gorm:"size:64;index"`
	Method       string    `json:"method" gorm:"size:16"`
	Path         string    `json:"path" gorm:"size:256"`
	Status       int       `json:"status"`
	LatencyMs    int64     `json:"latencyMs"`
	RequestBody  string    `json:"requestBody" gorm:"type:text"`
	ResponseBody string    `json:"responseBody" gorm:"type:text"`
	ErrorStack   string    `json:"errorStack" gorm:"type:text"`
	CreatedAt    time.Time `json:"createdAt"`
}

type PasswordResetToken struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"userId" gorm:"index;not null"`
	UserKind  string     `json:"userKind" gorm:"size:16;not null;default:admin"`
	Purpose   string     `json:"purpose" gorm:"size:16;not null;default:reset"`
	TokenHash string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"index;not null"`
	UsedAt    *time.Time `json:"usedAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

const (
	MailClassTransactional = "transactional"
	MailClassOperational   = "operational"
	MailClassMarketing     = "marketing"

	MailStatusQueued   = "queued"
	MailStatusSending  = "sending"
	MailStatusSent     = "sent"
	MailStatusFailed   = "failed"
	MailStatusDead     = "dead"
	MailStatusCanceled = "canceled"

	MailPriorityUrgent = 1
	MailPriorityNormal = 5
	MailPriorityLow    = 9

	CampaignDraft     = "draft"
	CampaignScheduled = "scheduled"
	CampaignRunning   = "running"
	CampaignPaused    = "paused"
	CampaignDone      = "done"

	AudienceOptedIn   = "opted_in"
	AudienceAllActive = "all_active"
)

type MailJob struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	CampaignID *uint      `json:"campaignId" gorm:"index"`
	Class      string     `json:"class" gorm:"size:16;index;not null"`
	Priority   int        `json:"priority" gorm:"not null;default:5;index"`
	UserID     *uint      `json:"userId" gorm:"index"`
	UserKind   string     `json:"userKind" gorm:"size:16"`
	ToEmail    string     `json:"toEmail" gorm:"size:128;not null"`
	Timezone   string     `json:"timezone" gorm:"size:64"`
	Subject    string     `json:"subject" gorm:"size:255;not null"`
	Body       string     `json:"body" gorm:"type:text"`
	Status     string     `json:"status" gorm:"size:16;index;not null;default:queued"`
	SendAfter  time.Time  `json:"sendAfter" gorm:"index;not null"`
	Attempts   int        `json:"attempts" gorm:"not null;default:0"`
	LastError  string     `json:"lastError" gorm:"size:512"`
	DedupeKey  string     `json:"dedupeKey" gorm:"size:128;index"`
	SentAt     *time.Time `json:"sentAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type MailCampaign struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Subject     string     `json:"subject" gorm:"size:255;not null"`
	Body        string     `json:"body" gorm:"type:text"`
	Audience    string     `json:"audience" gorm:"size:32;not null;default:opted_in"`
	Status      string     `json:"status" gorm:"size:16;index;not null;default:draft"`
	ScheduledAt *time.Time `json:"scheduledAt"`
	StartedAt   *time.Time `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt"`
	JobCount    int        `json:"jobCount" gorm:"not null;default:0"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type AuthSession struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"userId" gorm:"index;not null"`
	UserKind  string     `json:"userKind" gorm:"size:16;not null;default:admin"`
	JTI       string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	IP        string     `json:"ip" gorm:"size:64"`
	UserAgent string     `json:"userAgent" gorm:"size:512"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"index;not null"`
	RevokedAt *time.Time `json:"revokedAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (AuthSession) TableName() string { return "auth_sessions" }

type UserImportJob struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	ActorID      uint      `json:"actorId" gorm:"not null;index"`
	Kind         string    `json:"kind" gorm:"size:16;not null"`
	FileName     string    `json:"fileName" gorm:"size:255"`
	Status       string    `json:"status" gorm:"size:16;not null;index"`
	Total        int       `json:"total"`
	CreatedCount int       `json:"created"`
	FailedCount  int       `json:"failed"`
	Errors       string    `json:"errors" gorm:"type:text"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (UserImportJob) TableName() string { return "user_import_jobs" }

func AllModels() []any {
	return []any{
		&User{}, &webUserModel{}, &WebUserRole{},
		&Role{}, &Permission{}, &Department{},
		&NavMenu{},
		&DictType{}, &DictItem{}, &SysConfig{},
		&LoginLog{}, &OpLog{}, &APILog{}, &PasswordResetToken{},
		&MailJob{}, &MailCampaign{}, &AuthSession{}, &UserImportJob{},
	}
}
