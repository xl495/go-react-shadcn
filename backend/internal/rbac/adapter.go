package rbac

import (
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

type Rule struct {
	ID    uint   `gorm:"primaryKey"`
	Ptype string `gorm:"size:32;index"`
	V0    string `gorm:"size:128"`
	V1    string `gorm:"size:128"`
	V2    string `gorm:"size:32"`
	V3    string `gorm:"size:32"`
	V4    string `gorm:"size:32"`
	V5    string `gorm:"size:32"`
}

func (Rule) TableName() string { return "casbin_rule" }

type gormAdapter struct {
	db *gorm.DB
}

func newAdapter(db *gorm.DB) (*gormAdapter, error) {
	if err := db.AutoMigrate(&Rule{}); err != nil {
		return nil, err
	}
	return &gormAdapter{db: db}, nil
}

func (a *gormAdapter) LoadPolicy(m model.Model) error {
	var rules []Rule
	if err := a.db.Find(&rules).Error; err != nil {
		return err
	}
	for _, r := range rules {
		line := joinRule(r.Ptype, []string{r.V0, r.V1, r.V2, r.V3, r.V4, r.V5})
		if err := persist.LoadPolicyLine(line, m); err != nil {
			return err
		}
	}
	return nil
}

func (a *gormAdapter) SavePolicy(m model.Model) error {
	if err := a.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Rule{}).Error; err != nil {
		return err
	}
	var recs []Rule
	for ptype, ast := range m["p"] {
		for _, rule := range ast.Policy {
			recs = append(recs, toRule(ptype, rule))
		}
	}
	for ptype, ast := range m["g"] {
		for _, rule := range ast.Policy {
			recs = append(recs, toRule(ptype, rule))
		}
	}
	if len(recs) == 0 {
		return nil
	}
	return a.db.Create(&recs).Error
}

func (a *gormAdapter) AddPolicy(_ string, ptype string, rule []string) error {
	rec := toRule(ptype, rule)
	return a.db.Create(&rec).Error
}

func (a *gormAdapter) RemovePolicy(_ string, ptype string, rule []string) error {
	rec := toRule(ptype, rule)
	return a.db.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ? AND v3 = ? AND v4 = ? AND v5 = ?",
		rec.Ptype, rec.V0, rec.V1, rec.V2, rec.V3, rec.V4, rec.V5).Delete(&Rule{}).Error
}

func (a *gormAdapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	q := a.db.Where("ptype = ?", ptype)
	for i, v := range fieldValues {
		if v == "" {
			continue
		}
		col := [...]string{"v0", "v1", "v2", "v3", "v4", "v5"}[fieldIndex+i]
		q = q.Where(col+" = ?", v)
	}
	return q.Delete(&Rule{}).Error
}

func toRule(ptype string, rule []string) Rule {
	vals := [6]string{}
	for i := 0; i < len(rule) && i < 6; i++ {
		vals[i] = rule[i]
	}
	return Rule{Ptype: ptype, V0: vals[0], V1: vals[1], V2: vals[2], V3: vals[3], V4: vals[4], V5: vals[5]}
}

func joinRule(ptype string, vals []string) string {
	parts := []string{ptype}
	for _, v := range vals {
		if v == "" {
			break
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, ", ")
}
