package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

func countOrFail(c *gin.Context, q *gorm.DB, code int, msg string) (int64, bool) {
	var total int64
	if err := q.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, code, msg)
		return 0, false
	}
	return total, true
}

func parseQueryUint(c *gin.Context, key string) uint {
	v := strings.TrimSpace(c.Query(key))
	if v == "" {
		return 0
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil || n == 0 {
		return 0
	}
	return uint(n)
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "!", "!!")
	s = strings.ReplaceAll(s, "%", "!%")
	s = strings.ReplaceAll(s, "_", "!_")
	return s
}

func applyEqual(q *gorm.DB, col, v string) *gorm.DB {
	v = strings.TrimSpace(v)
	if v == "" {
		return q
	}
	return q.Where(col+" = ?", v)
}

func applyContains(q *gorm.DB, kw string, cols ...string) *gorm.DB {
	kw = strings.TrimSpace(kw)
	if kw == "" || len(cols) == 0 {
		return q
	}
	if utf8.RuneCountInString(kw) < 2 {
		return q.Where("1 = 0")
	}
	pat := "%" + escapeLike(kw) + "%"
	parts := make([]string, len(cols))
	args := make([]any, 0, len(cols))
	for i, col := range cols {
		parts[i] = col + " LIKE ? ESCAPE '!'"
		args = append(args, pat)
	}
	return q.Where("("+strings.Join(parts, " OR ")+")", args...)
}

func applyPrefix(q *gorm.DB, kw, col string) *gorm.DB {
	kw = strings.TrimSpace(kw)
	if kw == "" || col == "" {
		return q
	}
	if utf8.RuneCountInString(kw) < 2 {
		return q.Where("1 = 0")
	}
	return q.Where(col+" LIKE ? ESCAPE '!'", escapeLike(kw)+"%")
}

func applyUserKeyword(q *gorm.DB, tbl, kw string) *gorm.DB {
	kw = strings.TrimSpace(kw)
	if kw == "" {
		return q
	}
	if utf8.RuneCountInString(kw) < 2 {
		return q.Where("1 = 0")
	}
	pat := "%" + escapeLike(kw) + "%"
	like := " LIKE ? ESCAPE '!'"
	dict := " IN (SELECT value FROM dict_items WHERE type_code = ? AND label LIKE ? ESCAPE '!')"
	dept := tbl + ".department_id IN (SELECT id FROM departments WHERE name" + like + " OR code" + like + " OR leader" + like + ")"
	if match, ok := userFTSMatch(q, tbl, kw); ok {
		sql := strings.Join([]string{
			tbl + ".id IN (SELECT rowid FROM " + tbl + "_fts WHERE " + tbl + "_fts MATCH ?)",
			tbl + ".title" + like,
			tbl + ".remark" + like,
			tbl + ".last_login_ip" + like,
			dept,
			tbl + ".gender" + dict,
			tbl + ".status" + dict,
		}, " OR ")
		return q.Where("("+sql+")",
			match, pat, pat, pat,
			pat, pat, pat,
			seed.DictGender, pat,
			seed.DictUserStatus, pat,
		)
	}
	sql := strings.Join([]string{
		tbl + ".username" + like,
		tbl + ".nickname" + like,
		tbl + ".email" + like,
		tbl + ".phone" + like,
		tbl + ".title" + like,
		tbl + ".remark" + like,
		tbl + ".last_login_ip" + like,
		dept,
		tbl + ".gender" + dict,
		tbl + ".status" + dict,
	}, " OR ")
	return q.Where("("+sql+")",
		pat, pat, pat, pat, pat, pat, pat,
		pat, pat, pat,
		seed.DictGender, pat,
		seed.DictUserStatus, pat,
	)
}

func userFTSMatch(q *gorm.DB, tbl, kw string) (string, bool) {
	if q == nil || q.Statement == nil || q.Statement.ConnPool == nil {
		return "", false
	}
	if strings.IndexFunc(kw, func(r rune) bool {
		switch r {
		case '"', '*', ':', '(', ')', '{', '}', '[', ']', '^', '-', '+':
			return true
		}
		return false
	}) >= 0 {
		return "", false
	}
	var n int64
	if err := q.Session(&gorm.Session{}).Raw("SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','virtual') AND name = ?", tbl+"_fts").Scan(&n).Error; err != nil || n == 0 {
		return "", false
	}
	return `"` + kw + `"*`, true
}
