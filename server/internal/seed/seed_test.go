package seed

import (
	"path/filepath"
	"testing"

	"go-react-shadcn/internal/migrate"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/store"
)

func TestEnsureRoleKeepsCustomDataScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.db")
	if err := migrate.Up(path); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	role, err := ensureRole(db, "Operator", RoleOperator, "ops")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&role).Update("data_scope", models.DataScopeSelf).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := ensureRole(db, "Operator", RoleOperator, "ops"); err != nil {
		t.Fatal(err)
	}
	var got models.Role
	if err := db.First(&got, role.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.DataScope != models.DataScopeSelf {
		t.Fatalf("dataScope=%q want %q", got.DataScope, models.DataScopeSelf)
	}
}
