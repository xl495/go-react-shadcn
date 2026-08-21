package httpserver

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/secretbox"
	"go-react-shadcn/internal/seed"
)

func TestConfigSecretsEncryptedAtRest(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var row models.SysConfig
	if err := app.DB.Where(`"key" = ?`, "auth.google_client_secret").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	upd := doJSON(t, app, http.MethodPut, "/api/v1/configs/"+itoa(row.ID), admin, map[string]string{
		"value": "google-secret-value", "name": row.Name, "group": row.Group, "remark": row.Remark,
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("update: %d %s", upd.Code, upd.Body.String())
	}
	if strings.Contains(upd.Body.String(), "google-secret-value") || strings.Contains(upd.Body.String(), "enc:v1:") {
		t.Fatalf("response leaked secret: %s", upd.Body.String())
	}
	if err := app.DB.First(&row, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !secretbox.IsSealed(row.Value) {
		t.Fatalf("want sealed, got %q", row.Value)
	}
	if secretbox.MustOpen(app.Cfg.JWTSecret, row.Value) != "google-secret-value" {
		t.Fatalf("decrypt=%q", secretbox.MustOpen(app.Cfg.JWTSecret, row.Value))
	}
	listed := doJSON(t, app, http.MethodGet, "/api/v1/configs?group=auth&pageSize=50", admin, nil)
	if strings.Contains(listed.Body.String(), "google-secret-value") || strings.Contains(listed.Body.String(), "enc:v1:") {
		t.Fatalf("list leaked secret or ciphertext: %s", listed.Body.String())
	}
	if !strings.Contains(listed.Body.String(), mailer.SecretMask) {
		t.Fatalf("expected mask: %s", listed.Body.String())
	}
}

func TestDepartmentCodeSyncsUserDepartment(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var dept models.Department
	if err := app.DB.Where("code = ?", "tech").First(&dept).Error; err != nil {
		t.Fatal(err)
	}
	created := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "deptlink", "password": "Dept-pass1", "status": "active", "kind": "admin",
		"department": "tech",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create user: %d %s", created.Code, created.Body.String())
	}
	var user models.User
	if err := app.DB.Where("username = ?", "deptlink").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.DepartmentID == nil || *user.DepartmentID != dept.ID {
		t.Fatalf("department_id=%v want %d", user.DepartmentID, dept.ID)
	}

	filled := models.User{DepartmentID: &dept.ID}
	app.applyDepartmentLink(&filled)
	if filled.Department != "tech" {
		t.Fatalf("fill from id: %q", filled.Department)
	}

	upd := doJSON(t, app, http.MethodPut, "/api/v1/departments/"+strconv.FormatUint(uint64(dept.ID), 10), admin, map[string]any{
		"name": "Tech renamed", "code": "tech-renamed", "status": "active", "sort": dept.Sort,
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("update dept: %d %s", upd.Code, upd.Body.String())
	}
	if err := app.DB.Where("username = ?", "deptlink").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.Department != "tech-renamed" {
		t.Fatalf("user.department=%q after rename", user.Department)
	}

	renamed := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?department=tech-renamed", admin, nil))
	if !usernamesContain(renamed, "deptlink") {
		t.Fatalf("filter by renamed department: %+v", usernames(renamed))
	}
	stale := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?department=tech", admin, nil))
	if usernamesContain(stale, "deptlink") {
		t.Fatalf("old department code still matched: %+v", usernames(stale))
	}

	hq := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "hquser", "password": "Dept-pass1", "status": "active", "kind": "admin",
		"department": "hq",
	})
	if hq.Code != http.StatusOK {
		t.Fatalf("assign hq: %d %s", hq.Code, hq.Body.String())
	}
	byName := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?q="+url.QueryEscape("总部"), admin, nil))
	if !usernamesContain(byName, "hquser") {
		t.Fatalf("search department name: %+v", usernames(byName))
	}

	lab := doJSON(t, app, http.MethodPost, "/api/v1/departments", admin, map[string]any{
		"name": "Overflow Lab", "code": "ov-lab", "status": "active",
	})
	if lab.Code != http.StatusOK {
		t.Fatalf("create dept: %d %s", lab.Code, lab.Body.String())
	}
	dict := lookupDictValues(t, app, admin, seed.DictDepartment)
	if !dict["ov-lab"] {
		t.Fatalf("dict missing new department: %v", dict)
	}
	if dict["tech"] {
		t.Fatalf("renamed tech should leave dict: %v", dict)
	}
}

func usernamesContain(users []userDTO, name string) bool {
	for _, u := range users {
		if u.Username == name {
			return true
		}
	}
	return false
}
