package httpserver

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestNavMenuIsSingleTable(t *testing.T) {
	app := testApp(t)
	var n int64
	if err := app.DB.Model(&models.NavMenu{}).Count(&n).Error; err != nil || n == 0 {
		t.Fatalf("nav_menu empty: n=%d err=%v", n, err)
	}
	var adminN, webN int64
	_ = app.DB.Model(&models.NavMenu{}).Where("audience = ?", models.NavAudienceAdmin).Count(&adminN).Error
	_ = app.DB.Model(&models.NavMenu{}).Where("audience = ?", models.NavAudienceWeb).Count(&webN).Error
	if adminN == 0 || webN == 0 {
		t.Fatalf("audience split admin=%d web=%d", adminN, webN)
	}
}

func TestBatchConfigsAndExportAndRevoke(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	var row models.SysConfig
	if err := app.DB.Where(`"key" = ?`, "app.name").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	saved := doJSON(t, app, http.MethodPut, "/api/v1/configs/batch", admin, map[string]any{
		"items": []map[string]any{{"id": row.ID, "value": "Latch Batch", "name": row.Name}},
	})
	if saved.Code != http.StatusOK {
		t.Fatalf("batch: %d %s", saved.Code, saved.Body.String())
	}

	csv := doJSON(t, app, http.MethodGet, "/api/v1/users/export", admin, nil)
	if csv.Code != http.StatusOK || !strings.Contains(csv.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("export users: %d %s", csv.Code, csv.Body.String())
	}
	if !strings.Contains(csv.Body.String(), "admin") {
		t.Fatalf("export missing admin: %s", csv.Body.String())
	}

	logs := doJSON(t, app, http.MethodGet, "/api/v1/logs/export", admin, nil)
	if logs.Code != http.StatusOK || !strings.Contains(logs.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("export logs: %d %s", logs.Code, logs.Body.String())
	}

	var viewer models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	tok := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	rev := doJSON(t, app, http.MethodPost, "/api/v1/users/"+formatUint(viewer.ID)+"/revoke", admin, nil)
	if rev.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", rev.Code, rev.Body.String())
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", tok, nil)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("old token after revoke: %d %s", me.Code, me.Body.String())
	}

	dash := doJSON(t, app, http.MethodGet, "/api/v1/dashboard/stats", admin, nil)
	if dash.Code != http.StatusOK || !strings.Contains(dash.Body.String(), "mailQueued") {
		t.Fatalf("dashboard: %d %s", dash.Code, dash.Body.String())
	}
}

func TestAuthSessionsListAndKickOne(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	first := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	second := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	var viewer models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	listed := doJSON(t, app, http.MethodGet, "/api/v1/users/"+formatUint(viewer.ID)+"/sessions", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("sessions: %d %s", listed.Code, listed.Body.String())
	}
	var rows []models.AuthSession
	if err := json.Unmarshal(decodeEnv(t, listed).Data, &rows); err != nil || len(rows) < 2 {
		t.Fatalf("session rows: %s", listed.Body.String())
	}
	oldest := rows[len(rows)-1]
	kicked := doJSON(t, app, http.MethodDelete, "/api/v1/users/"+formatUint(viewer.ID)+"/sessions/"+formatUint(oldest.ID), admin, nil)
	if kicked.Code != http.StatusOK {
		t.Fatalf("kick: %d %s", kicked.Code, kicked.Body.String())
	}
	stale := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", first, nil)
	if stale.Code != http.StatusUnauthorized {
		t.Fatalf("kicked session still live: %d %s", stale.Code, stale.Body.String())
	}
	alive := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", second, nil)
	if alive.Code != http.StatusOK {
		t.Fatalf("other session died: %d %s", alive.Code, alive.Body.String())
	}
}

func TestImportUsersCSVAsync(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	csvBody := "username,password,nickname,email,phone,status,department,kind\ncsvuser,csvuser12,CSV User,csvuser@example.com,,active,,admin\n"
	posted := doMultipart(t, app, "/api/v1/users/import?kind=admin", admin, "file", "users.csv", []byte(csvBody))
	if posted.Code != http.StatusOK {
		t.Fatalf("import: %d %s", posted.Code, posted.Body.String())
	}
	var job models.UserImportJob
	if err := json.Unmarshal(decodeEnv(t, posted).Data, &job); err != nil || job.ID == 0 {
		t.Fatalf("job: %s", posted.Body.String())
	}
	var got models.UserImportJob
	for i := 0; i < 40; i++ {
		listed := doJSON(t, app, http.MethodGet, "/api/v1/users/import-jobs/"+formatUint(job.ID), admin, nil)
		if listed.Code != http.StatusOK {
			t.Fatalf("job get: %d %s", listed.Code, listed.Body.String())
		}
		if err := json.Unmarshal(decodeEnv(t, listed).Data, &got); err != nil {
			t.Fatal(err)
		}
		if got.Status == "done" || got.Status == "failed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got.Status != "done" || got.CreatedCount != 1 {
		t.Fatalf("job %+v", got)
	}
	var row models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", "csvuser").First(&row).Error; err != nil {
		t.Fatal(err)
	}

	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	denied := doMultipart(t, app, "/api/v1/users/import", viewer, "file", "users.csv", []byte(csvBody))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("viewer import: %d %s", denied.Code, denied.Body.String())
	}
}

func TestBatchConfigsAllowedWithUpdatePerm(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	var listPerm, updatePerm, batchPerm models.Permission
	if err := app.DB.Where("code = ?", "config:list").First(&listPerm).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Where("code = ?", "config:update").First(&updatePerm).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Where("code = ?", "config:batch").First(&batchPerm).Error; err != nil {
		t.Fatal(err)
	}

	roles := decodeRolePage(t, doJSON(t, app, http.MethodGet, "/api/v1/roles", admin, nil))
	var viewer roleDTO
	for _, r := range roles {
		if r.Code == seed.RoleViewer {
			viewer = r
			break
		}
	}
	if viewer.ID == 0 {
		t.Fatal("missing viewer role")
	}
	ids := append(append([]uint{}, viewer.PermissionIDs...), listPerm.ID, updatePerm.ID)
	for _, id := range ids {
		if id == batchPerm.ID {
			t.Fatal("viewer fixture already has config:batch")
		}
	}
	assign := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+itoa(viewer.ID)+"/permissions", admin, map[string]any{
		"permissionIds": ids,
	})
	if assign.Code != http.StatusOK {
		t.Fatalf("assign: %d %s", assign.Code, assign.Body.String())
	}

	var row models.SysConfig
	if err := app.DB.Where(`"key" = ?`, "app.name").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	tok := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	saved := doJSON(t, app, http.MethodPut, "/api/v1/configs/batch", tok, map[string]any{
		"items": []map[string]any{{"id": row.ID, "value": "gra-batch", "name": row.Name}},
	})
	if saved.Code != http.StatusOK {
		t.Fatalf("update without batch perm: %d %s", saved.Code, saved.Body.String())
	}

	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	denied := doJSON(t, app, http.MethodPut, "/api/v1/configs/batch", operator, map[string]any{
		"items": []map[string]any{{"id": row.ID, "value": "nope", "name": row.Name}},
	})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("operator without update should 403, got %d %s", denied.Code, denied.Body.String())
	}
}

func doMultipart(t *testing.T, app *App, path, token, field, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)
	return rec
}
