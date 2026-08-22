package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/totp"
)

func TestCoverageBoostAdminSurfaces(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	if w := doJSON(t, app, http.MethodGet, "/openapi.yaml", "", nil); w.Code != http.StatusOK {
		t.Fatalf("openapi: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/nav-menus", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list nav: %d %s", w.Code, w.Body.String())
	}
	createdNav := doJSON(t, app, http.MethodPost, "/api/v1/nav-menus", admin, map[string]any{
		"audience": "admin", "name": "Extra", "code": "extra:menu", "routePath": "/extra",
		"component": "DashboardPage", "icon": "Radio", "sort": 99, "permCode": "dashboard:read",
	})
	if createdNav.Code != http.StatusOK {
		t.Fatalf("create nav: %d %s", createdNav.Code, createdNav.Body.String())
	}
	var nav models.NavMenu
	if err := json.Unmarshal(decodeEnv(t, createdNav).Data, &nav); err != nil {
		t.Fatal(err)
	}
	updNav := doJSON(t, app, http.MethodPut, "/api/v1/nav-menus/"+itoa(nav.ID), admin, map[string]any{
		"name": "Extra 2", "sort": 98,
	})
	if updNav.Code != http.StatusOK {
		t.Fatalf("update nav: %d %s", updNav.Code, updNav.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/nav-menus/"+itoa(nav.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("delete nav: %d %s", w.Code, w.Body.String())
	}

	if w := doJSON(t, app, http.MethodGet, "/api/v1/notifications/unread-count", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("unread: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/notifications/read-all", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("read-all: %d %s", w.Code, w.Body.String())
	}

	if w := doJSON(t, app, http.MethodGet, "/api/v1/logs/api", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("api logs: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/logs", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("clear logs: %d %s", w.Code, w.Body.String())
	}

	dict := doJSON(t, app, http.MethodPost, "/api/v1/dicts", admin, map[string]string{"code": "cov_level", "name": "覆盖"})
	if dict.Code != http.StatusOK {
		t.Fatalf("dict: %d %s", dict.Code, dict.Body.String())
	}
	var typ models.DictType
	if err := json.Unmarshal(decodeEnv(t, dict).Data, &typ); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/dicts/"+itoa(typ.ID), admin, map[string]string{"name": "覆盖2"}); w.Code != http.StatusOK {
		t.Fatalf("upd dict: %d %s", w.Code, w.Body.String())
	}
	item := doJSON(t, app, http.MethodPost, "/api/v1/dicts/"+itoa(typ.ID)+"/items", admin, map[string]any{"label": "A", "value": "a", "sort": 1})
	if item.Code != http.StatusOK {
		t.Fatalf("item: %d %s", item.Code, item.Body.String())
	}
	var di models.DictItem
	if err := json.Unmarshal(decodeEnv(t, item).Data, &di); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts/"+itoa(typ.ID)+"/items", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list items: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/dict-items/"+itoa(di.ID), admin, map[string]any{"label": "B", "value": "b"}); w.Code != http.StatusOK {
		t.Fatalf("upd item: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/dict-items/"+itoa(di.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del item: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/dicts/"+itoa(typ.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del dict: %d %s", w.Code, w.Body.String())
	}

	cfg := doJSON(t, app, http.MethodPost, "/api/v1/configs", admin, map[string]string{
		"key": "app.cov_flag", "value": "1", "name": "cov", "group": "app",
	})
	if cfg.Code != http.StatusOK {
		t.Fatalf("cfg: %d %s", cfg.Code, cfg.Body.String())
	}
	var saved models.SysConfig
	if err := json.Unmarshal(decodeEnv(t, cfg).Data, &saved); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/configs/"+itoa(saved.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del cfg: %d %s", w.Code, w.Body.String())
	}

	role := doJSON(t, app, http.MethodPost, "/api/v1/roles", admin, map[string]any{
		"name": "cov-role", "code": "cov-role", "dataScope": "self", "permissionIds": []uint{},
	})
	if role.Code != http.StatusOK {
		t.Fatalf("role: %d %s", role.Code, role.Body.String())
	}
	var createdRole roleDTO
	if err := json.Unmarshal(decodeEnv(t, role).Data, &createdRole); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/roles/"+itoa(createdRole.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del role: %d %s", w.Code, w.Body.String())
	}

	perm := doJSON(t, app, http.MethodPost, "/api/v1/permissions", admin, map[string]any{
		"name": "cov perm", "code": "cov:perm", "path": "/api/v1/cov", "method": "GET", "kind": "api",
	})
	if perm.Code != http.StatusOK {
		t.Fatalf("perm: %d %s", perm.Code, perm.Body.String())
	}
	var createdPerm permissionDTO
	if err := json.Unmarshal(decodeEnv(t, perm).Data, &createdPerm); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/permissions/"+itoa(createdPerm.ID), admin, map[string]any{
		"name": "cov perm 2", "code": "cov:perm", "path": "/api/v1/cov", "method": "GET", "kind": "api",
	}); w.Code != http.StatusOK {
		t.Fatalf("upd perm: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/permissions/"+itoa(createdPerm.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del perm: %d %s", w.Code, w.Body.String())
	}

	user := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "cov-user", "password": "cov-user-1a", "roleIds": []uint{},
	})
	if user.Code != http.StatusOK {
		t.Fatalf("user: %d %s", user.Code, user.Body.String())
	}
	var createdUser userDTO
	if err := json.Unmarshal(decodeEnv(t, user).Data, &createdUser); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/users/"+itoa(createdUser.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del user: %d %s", w.Code, w.Body.String())
	}

	camp := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns", admin, map[string]string{
		"name": "cov", "subject": "hi", "body": "<p>x</p>", "audience": "opted_in",
	})
	if camp.Code != http.StatusOK {
		t.Fatalf("campaign: %d %s", camp.Code, camp.Body.String())
	}
	var createdCamp models.MailCampaign
	if err := json.Unmarshal(decodeEnv(t, camp).Data, &createdCamp); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/mail/campaigns/"+itoa(createdCamp.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del campaign: %d %s", w.Code, w.Body.String())
	}

	setup := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/setup", viewer, map[string]string{})
	if setup.Code != http.StatusOK {
		t.Fatalf("totp setup: %d %s", setup.Code, setup.Body.String())
	}
	var enroll struct {
		Secret string `json:"secret"`
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(decodeEnv(t, setup).Data, &enroll); err != nil {
		t.Fatal(err)
	}
	code, err := totp.Code(enroll.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	confirm := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/confirm", viewer, map[string]string{
		"ticket": enroll.Ticket, "code": code,
	})
	if confirm.Code != http.StatusOK {
		t.Fatalf("totp confirm: %d %s", confirm.Code, confirm.Body.String())
	}
	code2, err := totp.Code(enroll.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/disable", viewer, map[string]string{"code": code2}); w.Code != http.StatusOK {
		t.Fatalf("totp disable: %d %s", w.Code, w.Body.String())
	}

	app.sessions.Sweep()
	app.sweepTotpTickets(time.Now().Add(time.Hour))
}
