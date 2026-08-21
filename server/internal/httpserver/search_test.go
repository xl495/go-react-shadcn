package httpserver

import (
	"net/http"
	"net/url"
	"testing"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestUserListSearchAndFilters(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	byPhone := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?q=13800000003", admin, nil))
	if len(byPhone) != 1 || byPhone[0].Username != seed.ViewerUsername {
		t.Fatalf("phone search: %+v", usernames(byPhone))
	}
	byTitle := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?q="+url.QueryEscape("负责人"), admin, nil))
	if len(byTitle) != 1 || byTitle[0].Username != seed.AdminUsername {
		t.Fatalf("title search: %+v", usernames(byTitle))
	}
	byGender := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?gender=female", admin, nil))
	if len(byGender) != 1 || byGender[0].Username != seed.ViewerUsername {
		t.Fatalf("gender filter: %+v", usernames(byGender))
	}
	byDept := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?department=ops", admin, nil))
	if len(byDept) != 1 || byDept[0].Username != seed.OperatorUsername {
		t.Fatalf("department filter: %+v", usernames(byDept))
	}
	byStatus := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?status=active&pageSize=50", admin, nil))
	if len(byStatus) < 3 {
		t.Fatalf("status=active want >=3 got %d", len(byStatus))
	}

	var viewerRole models.Role
	if err := app.DB.Where("code = ?", seed.RoleViewer).First(&viewerRole).Error; err != nil {
		t.Fatal(err)
	}
	byRole := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?roleId="+itoa(viewerRole.ID), admin, nil))
	if len(byRole) != 1 || byRole[0].Username != seed.ViewerUsername {
		t.Fatalf("role filter: %+v", usernames(byRole))
	}

	combined := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?q=admin&gender=female", admin, nil))
	if len(combined) != 0 {
		t.Fatalf("q AND gender should not leak via OR, got %+v", usernames(combined))
	}
	byDeptLabel := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?q="+url.QueryEscape("运营"), admin, nil))
	if len(byDeptLabel) != 1 || byDeptLabel[0].Username != seed.OperatorUsername {
		t.Fatalf("department label search: %+v", usernames(byDeptLabel))
	}

	wild := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?q=%25", admin, nil))
	if len(wild) != 0 {
		t.Fatalf("LIKE wildcard should be escaped, got %+v", usernames(wild))
	}

	staff := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?kind=admin&pageSize=50", admin, nil))
	webUsers := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?kind=web&pageSize=50", admin, nil))
	if len(staff) < 3 {
		t.Fatalf("kind=admin want >=3 staff, got %+v", usernames(staff))
	}
	for _, u := range staff {
		if u.Kind != "admin" {
			t.Fatalf("kind=admin leaked %s kind=%s", u.Username, u.Kind)
		}
		if u.Username == seed.MemberUsername {
			t.Fatal("webuser must not appear in staff list")
		}
	}
	if len(webUsers) != 1 || webUsers[0].Username != seed.MemberUsername || webUsers[0].Kind != "web" {
		t.Fatalf("kind=web: %+v", webUsers)
	}

	id, answer, _ := issueCaptcha(t, app)
	blocked := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username":    seed.MemberUsername,
		"password":    seed.MemberPassword,
		"captchaId":   id,
		"captchaCode": answer,
		"client":      "admin",
	})
	if blocked.Code != http.StatusUnauthorized {
		t.Fatalf("web user on admin login want 401 got %d %s", blocked.Code, blocked.Body.String())
	}
	id, answer, _ = issueCaptcha(t, app)
	webLogin := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username":    seed.MemberUsername,
		"password":    seed.MemberPassword,
		"captchaId":   id,
		"captchaCode": answer,
		"client":      "web",
	})
	if webLogin.Code != http.StatusOK {
		t.Fatalf("web user login want 200 got %d %s", webLogin.Code, webLogin.Body.String())
	}
}

func TestPermissionListSearch(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	listed := doJSON(t, app, http.MethodGet, "/api/v1/permissions?q=user:list&pageSize=50", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	items := decodePage[permissionDTO](t, listed).Items
	if len(items) != 1 || items[0].Code != "user:list" {
		t.Fatalf("q=user:list: %+v", permCodes(items))
	}

	menus := decodePage[permissionDTO](t, doJSON(t, app, http.MethodGet, "/api/v1/permissions?kind=menu&pageSize=200", admin, nil)).Items
	if len(menus) == 0 {
		t.Fatal("expected menu permissions")
	}
	for _, p := range menus {
		if p.Kind != seed.KindMenu {
			t.Fatalf("kind=menu leaked %s kind=%s", p.Code, p.Kind)
		}
	}
}

func TestListSearchOnRolesConfigsDicts(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	roles := decodePage[roleDTO](t, doJSON(t, app, http.MethodGet, "/api/v1/roles?q=admin", admin, nil)).Items
	if len(roles) != 1 || roles[0].Code != seed.RoleAdmin {
		t.Fatalf("roles q=admin: %+v", roles)
	}
	cfgs := decodePage[models.SysConfig](t, doJSON(t, app, http.MethodGet, "/api/v1/configs?q=mail.host", admin, nil)).Items
	if len(cfgs) != 1 || cfgs[0].Key != "mail.host" {
		t.Fatalf("configs q=mail.host: %+v", cfgs)
	}
	dicts := decodePage[models.DictType](t, doJSON(t, app, http.MethodGet, "/api/v1/dicts?q=sys_gender", admin, nil)).Items
	if len(dicts) != 1 || dicts[0].Code != seed.DictGender {
		t.Fatalf("dicts q=sys_gender: %+v", dicts)
	}
	depts := decodePage[models.Department](t, doJSON(t, app, http.MethodGet, "/api/v1/departments?q="+url.QueryEscape("运营"), admin, nil)).Items
	if len(depts) == 0 {
		t.Fatal("departments q=运营 expected a match")
	}
}

func TestSameUsernameAllowedOnSplitTables(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	created := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": seed.AdminUsername,
		"password": "web-admin-99",
		"status":   "active",
		"kind":     models.UserKindWeb,
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create web user with staff username: %d %s", created.Code, created.Body.String())
	}
	id, answer, _ := issueCaptcha(t, app)
	webLogin := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username":    seed.AdminUsername,
		"password":    "web-admin-99",
		"captchaId":   id,
		"captchaCode": answer,
		"client":      "web",
	})
	if webLogin.Code != http.StatusOK {
		t.Fatalf("web login same username: %d %s", webLogin.Code, webLogin.Body.String())
	}
	if loginOK(t, app, seed.AdminUsername, seed.AdminPassword) == "" {
		t.Fatal("admin login should still use admin_user")
	}
}

func usernames(rows []userDTO) []string {
	out := make([]string, 0, len(rows))
	for _, u := range rows {
		out = append(out, u.Username)
	}
	return out
}

func permCodes(rows []permissionDTO) []string {
	out := make([]string, 0, len(rows))
	for _, p := range rows {
		out = append(out, p.Code)
	}
	return out
}
