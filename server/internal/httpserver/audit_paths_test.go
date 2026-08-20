package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestLoginLockoutAfterFailures(t *testing.T) {
	app := testApp(t)
	user := seed.OperatorUsername
	for i := 0; i < 5; i++ {
		id, ans, _ := issueCaptcha(t, app)
		w := login(t, app, user, "wrong-password", id, ans)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: want 401 got %d %s", i+1, w.Code, w.Body.String())
		}
	}
	id, ans, _ := issueCaptcha(t, app)
	w := login(t, app, user, seed.OperatorPassword, id, ans)
	if w.Code != http.StatusForbidden {
		t.Fatalf("locked login want 403 got %d %s", w.Code, w.Body.String())
	}
	env := decodeEnv(t, w)
	if env.Code != CodeAccountLocked {
		t.Fatalf("code=%d want %d", env.Code, CodeAccountLocked)
	}
}

func TestTokenVersionRevokesOldJWT(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	okPwd := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", token, map[string]string{
		"oldPassword": seed.ViewerPassword,
		"newPassword": "viewer-pass-9",
	})
	if okPwd.Code != http.StatusOK {
		t.Fatalf("password change: %d %s", okPwd.Code, okPwd.Body.String())
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", token, nil)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("stale token want 401 got %d %s", me.Code, me.Body.String())
	}
}

func TestUserListHonorsDataScope(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)

	adminUsers := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?pageSize=50", admin, nil))
	opUsers := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?pageSize=50", operator, nil))
	if len(adminUsers) < 3 {
		t.Fatalf("admin should see all seed users, got %d", len(adminUsers))
	}
	if len(opUsers) == 0 || len(opUsers) >= len(adminUsers) {
		t.Fatalf("operator data scope should be narrower than admin: admin=%d operator=%d", len(adminUsers), len(opUsers))
	}
	for _, u := range opUsers {
		if u.Department != "" && u.Department != "ops" && u.Username != seed.OperatorUsername {
			t.Fatalf("operator saw out-of-scope user %s dept=%s", u.Username, u.Department)
		}
	}
}

func TestMenuTreeFollowsPermissions(t *testing.T) {
	app := testApp(t)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	w := doJSON(t, app, http.MethodGet, "/api/v1/auth/menus", viewer, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("viewer menus: %d %s", w.Code, w.Body.String())
	}
	var menus []menuNode
	if err := json.Unmarshal(decodeEnv(t, w).Data, &menus); err != nil {
		t.Fatalf("decode menus: %v", err)
	}
	if len(flattenMenuTest(menus)) == 0 {
		t.Fatal("viewer should have dashboard menu")
	}
	for _, m := range flattenMenuTest(menus) {
		if m.Code == "user:list" {
			t.Fatal("viewer must not see users menu")
		}
	}

	aw := doJSON(t, app, http.MethodGet, "/api/v1/auth/menus", admin, nil)
	var adminMenus []menuNode
	if err := json.Unmarshal(decodeEnv(t, aw).Data, &adminMenus); err != nil {
		t.Fatal(err)
	}
	foundDept := false
	var org *menuNode
	for i := range adminMenus {
		if adminMenus[i].Code == "org:menu" {
			org = &adminMenus[i]
		}
	}
	if org == nil {
		t.Fatalf("admin menus missing org group: %+v", adminMenus)
	}
	for _, m := range flattenMenuTest(adminMenus) {
		if m.Code == "dept:list" && m.Component == "DepartmentsPage" {
			foundDept = true
		}
	}
	if !foundDept {
		t.Fatalf("admin menus missing department page: %+v", adminMenus)
	}
	foundUsers := false
	for _, c := range org.Children {
		if c.Code == "user:list" {
			foundUsers = true
		}
	}
	if !foundUsers {
		t.Fatalf("user:list should nest under org:menu: %+v", org.Children)
	}

	op := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	ow := doJSON(t, app, http.MethodGet, "/api/v1/auth/menus", op, nil)
	var opMenus []menuNode
	if err := json.Unmarshal(decodeEnv(t, ow).Data, &opMenus); err != nil {
		t.Fatal(err)
	}
	var opOrg *menuNode
	for i := range opMenus {
		if opMenus[i].Code == "org:menu" {
			opOrg = &opMenus[i]
		}
	}
	if opOrg == nil {
		t.Fatalf("operator should still see org group via ancestors: %+v", opMenus)
	}
}

func flattenMenuTest(nodes []menuNode) []menuNode {
	var out []menuNode
	for _, n := range nodes {
		out = append(out, n)
		if len(n.Children) > 0 {
			out = append(out, flattenMenuTest(n.Children)...)
		}
	}
	return out
}

func TestPurgeOldLogs(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	old := time.Now().AddDate(0, 0, -40)
	if err := app.DB.Create(&models.OpLog{
		Username: "stale",
		Module:   "user",
		Action:   "delete",
		Method:   "DELETE",
		Path:     "/api/v1/users/9",
		Status:   200,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Model(&models.OpLog{}).Where("username = ?", "stale").Update("created_at", old).Error; err != nil {
		t.Fatal(err)
	}
	w := doJSON(t, app, http.MethodPost, "/api/v1/logs/purge?days=30", admin, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("purge: %d %s", w.Code, w.Body.String())
	}
	var n int64
	app.DB.Model(&models.OpLog{}).Where("username = ?", "stale").Count(&n)
	if n != 0 {
		t.Fatalf("stale log still present: %d", n)
	}
}
