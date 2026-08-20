package httpserver

import (
	"testing"

	"go-react-shadcn/internal/models"
)

func TestIncludeMenuAncestorsPromotesParents(t *testing.T) {
	orgID := uint(1)
	parent := models.Permission{ID: 1, Code: "org:menu", Kind: "menu"}
	child := models.Permission{ID: 2, Code: "user:list", Kind: "menu", ParentID: &orgID}
	got := includeMenuAncestors([]models.Permission{parent, child}, []models.Permission{child})
	if len(got) != 2 {
		t.Fatalf("want 2 perms, got %d", len(got))
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
	perms := []models.Permission{
		{ID: 1, Code: "org:menu", Kind: "menu", Name: "组织"},
		{ID: 2, Code: "user:list", Kind: "menu", Name: "用户", ParentID: &orgID},
		{ID: 3, Code: "dashboard:read", Kind: "menu", Name: "仪表盘"},
	}
	tree := buildMenuTree(perms, nil)
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
