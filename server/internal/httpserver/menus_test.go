package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/seed"
)

func TestIncludeMenuAncestorsPromotesParents(t *testing.T) {
	orgID := uint(1)
	parent := models.NavMenu{ID: 1, Code: "org:menu"}
	child := models.NavMenu{ID: 2, Code: "user:list", ParentID: &orgID, PermCode: "user:list"}
	got := includeMenuAncestors([]models.NavMenu{parent, child}, []models.NavMenu{child})
	if len(got) != 2 {
		t.Fatalf("want 2 menus, got %d", len(got))
	}
	codes := map[string]bool{}
	for _, p := range got {
		codes[p.Code] = true
	}
	if !codes["org:menu"] || !codes["user:list"] {
		t.Fatalf("missing ancestor or child: %+v", got)
	}
}

func TestBuildMenuTreeNestsChildren(t *testing.T) {
	orgID := uint(1)
	rows := []models.NavMenu{
		{ID: 1, Code: "org:menu", Name: "组织"},
		{ID: 2, Code: "user:list", Name: "用户", ParentID: &orgID},
		{ID: 3, Code: "dashboard:read", Name: "仪表盘"},
	}
	tree := buildMenuTree(rows, nil)
	if len(tree) != 2 {
		t.Fatalf("want 2 roots, got %d %+v", len(tree), tree)
	}
	var org *menuNode
	for i := range tree {
		if tree[i].Code == "org:menu" {
			org = &tree[i]
		}
	}
	if org == nil {
		t.Fatalf("missing org group: %+v", tree)
	}
	if len(org.Children) != 1 || org.Children[0].Code != "user:list" {
		t.Fatalf("want user:list under org, got %+v", org.Children)
	}
}

func TestAdminAndWebMenusAreIsolated(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	adminTree := decodeMenus(t, doJSON(t, app, http.MethodGet, "/api/v1/auth/menus", admin, nil))
	for _, m := range flattenMenuTest(adminTree) {
		if strings.HasPrefix(m.Code, "web:") {
			t.Fatalf("admin menus leaked web item %s", m.Code)
		}
	}

	webTree := decodeMenus(t, doJSON(t, app, http.MethodGet, "/api/v1/auth/web-menus", viewer, nil))
	codes := map[string]bool{}
	for _, m := range flattenMenuTest(webTree) {
		if m.Code == "user:list" || m.Code == "org:menu" {
			t.Fatalf("web menus leaked admin item %s", m.Code)
		}
		codes[m.Code] = true
	}
	if !codes["web:home"] || !codes["web:profile"] || !codes["web:password"] {
		t.Fatalf("viewer web menus missing pages: %+v", webTree)
	}

	hash, err := passwd.Hash("plainpass1")
	if err != nil {
		t.Fatal(err)
	}
	nobody := models.User{Username: "nomenu", PasswordHash: hash, Status: "active", Timezone: "Asia/Shanghai"}
	if err := app.DB.Create(&nobody).Error; err != nil {
		t.Fatal(err)
	}
	tok := loginOK(t, app, "nomenu", "plainpass1")
	empty := decodeMenus(t, doJSON(t, app, http.MethodGet, "/api/v1/auth/web-menus", tok, nil))
	if len(flattenMenuTest(empty)) != 0 {
		t.Fatalf("user without me:read should see no web menus, got %+v", empty)
	}
}

func decodeMenus(t *testing.T, w *httptest.ResponseRecorder) []menuNode {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("menus status=%d body=%s", w.Code, w.Body.String())
	}
	var menus []menuNode
	if err := json.Unmarshal(decodeEnv(t, w).Data, &menus); err != nil {
		t.Fatalf("decode menus: %v", err)
	}
	return menus
}
