package httpserver

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

const importUserMaxRows = 2000

func (a *App) handleImportUsers(c *gin.Context) {
	actor, okActor := a.loadActor(c)
	if !okActor {
		return
	}
	kind := queryUserKind(c)
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "csv file required")
		return
	}
	if file.Size > 1<<20 {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "csv must be 1B-1MB")
		return
	}
	src, err := file.Open()
	if err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "csv file required")
		return
	}
	defer func() { _ = src.Close() }()
	data, err := io.ReadAll(io.LimitReader(src, 1<<20+1))
	if err != nil || len(data) == 0 {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "csv file required")
		return
	}
	if len(data) > 1<<20 {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "csv must be 1B-1MB")
		return
	}
	job := models.UserImportJob{
		ActorID:  actor.ID,
		Kind:     kind,
		FileName: filepath.Base(file.Filename),
		Status:   "queued",
	}
	if err := a.DB.Create(&job).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListUsers, "failed to import users")
		return
	}
	payload := append([]byte(nil), data...)
	go a.runUserImport(job.ID, payload)
	ok(c, job)
}

func (a *App) handleGetUserImportJob(c *gin.Context) {
	id, okID := parseIDParam(c)
	if !okID {
		fail(c, http.StatusNotFound, 40410, "import job not found")
		return
	}
	claims := currentUser(c)
	if claims == nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	var job models.UserImportJob
	if err := a.DB.Where("id = ? AND actor_id = ?", id, claims.UserID).First(&job).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "import job not found")
		return
	}
	ok(c, job)
}

func (a *App) runUserImport(id uint, data []byte) {
	a.importMu.Lock()
	defer a.importMu.Unlock()
	var job models.UserImportJob
	if err := a.DB.First(&job, id).Error; err != nil {
		return
	}
	_ = a.DB.Model(&job).Updates(map[string]any{"status": "running", "updated_at": time.Now()}).Error
	created, failed, total, errs := a.importUsersCSV(data, job.Kind)
	status := "done"
	if created == 0 && (failed > 0 || total == 0) {
		status = "failed"
	}
	_ = a.DB.Model(&job).Updates(map[string]any{
		"status":        status,
		"total":         total,
		"created_count": created,
		"failed_count":  failed,
		"errors":        errs,
		"updated_at":    time.Now(),
	}).Error
	a.notify(models.UserKindAdmin, job.ActorID, "import", "用户导入完成", job.FileName+" "+status, "import", job.ID)
}

func (a *App) importUsersCSV(data []byte, defaultKind string) (created, failed, total int, errText string) {
	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil || len(records) == 0 {
		return 0, 0, 0, "invalid csv"
	}
	header := records[0]
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	idx := csvIndex(header)
	usernameCol, okUser := idx["username"]
	passwordCol, okPass := idx["password"]
	if !okUser || !okPass {
		return 0, 0, 0, "csv must include username and password columns"
	}
	var lines []string
	for i, rec := range records[1:] {
		if i >= importUserMaxRows {
			lines = appendError(lines, i+2, fmt.Sprintf("skipped, max %d rows", importUserMaxRows))
			break
		}
		if csvRowEmpty(rec) {
			continue
		}
		total++
		username := csvCell(rec, usernameCol)
		password := csvCell(rec, passwordCol)
		if username == "" || password == "" {
			failed++
			lines = appendError(lines, i+2, "username and password required")
			continue
		}
		kind := defaultKind
		if col, ok := idx["kind"]; ok {
			if v := csvCell(rec, col); v != "" {
				if v != models.UserKindAdmin && v != models.UserKindWeb {
					failed++
					lines = appendError(lines, i+2, "invalid user kind")
					continue
				}
				kind = v
			}
		}
		if err := a.passwordIssue(password, username); err != nil {
			failed++
			lines = appendError(lines, i+2, err.Error())
			continue
		}
		status := "active"
		if col, ok := idx["status"]; ok {
			if v := csvCell(rec, col); v != "" {
				status = v
			}
		}
		nickname := csvCellAt(rec, idx, "nickname")
		email := csvCellAt(rec, idx, "email")
		phone := csvCellAt(rec, idx, "phone")
		gender := csvCellAt(rec, idx, "gender")
		department := csvCellAt(rec, idx, "department")
		if !a.dictValueOK(seed.DictUserStatus, status) ||
			!a.dictValueOK(seed.DictGender, gender) {
			failed++
			lines = appendError(lines, i+2, "invalid dictionary value")
			continue
		}
		if !a.departmentCodeOK(department) {
			failed++
			lines = appendError(lines, i+2, "invalid department")
			continue
		}
		if email != "" && a.emailTaken(kind, email, 0) {
			failed++
			lines = appendError(lines, i+2, "email already exists")
			continue
		}
		hash, err := passwd.Hash(password)
		if err != nil {
			failed++
			lines = appendError(lines, i+2, "failed to hash password")
			continue
		}
		roles, err := a.defaultRolesForKind(kind, nil)
		if err != nil {
			failed++
			lines = appendError(lines, i+2, "failed to assign role")
			continue
		}
		user := models.User{
			Username: username, PasswordHash: hash, Status: status,
			Nickname: nickname, Email: email, Phone: phone,
			Gender: gender, Department: department,
			Timezone: mailer.DefaultTimezone, MarketingOptIn: false, EmailVerified: true, Kind: kind,
		}
		a.applyDepartmentLink(&user)
		if err := a.withTx(func(tx *gorm.DB) error {
			if err := models.Accounts(tx, kind).Create(&user).Error; err != nil {
				return err
			}
			return models.ReplaceUserRoles(tx, kind, user.ID, roles)
		}); err != nil {
			failed++
			if isUniqueViolation(err) {
				lines = appendError(lines, i+2, "username already exists")
			} else {
				lines = appendError(lines, i+2, "failed to create user")
			}
			continue
		}
		if err := seed.SyncUserRoles(a.Enforcer, seed.CasbinSub(user.Kind, user.ID), roles); err != nil {
			failed++
			lines = appendError(lines, i+2, "failed to sync rbac")
			continue
		}
		created++
	}
	return created, failed, total, strings.Join(lines, "\n")
}

func (a *App) passwordIssue(plain, username string) error {
	if a.Cfg.DevMode {
		return passwd.Check(plain, username)
	}
	return passwd.CheckProduction(plain, username)
}

func csvIndex(header []string) map[string]int {
	out := map[string]int{}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if key != "" {
			out[key] = i
		}
	}
	return out
}

func csvCell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func csvCellAt(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok {
		return ""
	}
	return csvCell(row, i)
}

func csvRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func appendError(lines []string, row int, msg string) []string {
	if len(lines) >= 30 {
		return lines
	}
	if !utf8.ValidString(msg) {
		msg = "invalid row"
	}
	return append(lines, fmt.Sprintf("row %d: %s", row, msg))
}
