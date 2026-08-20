package httpserver

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

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
	pat := "%" + escapeLike(kw) + "%"
	parts := make([]string, len(cols))
	args := make([]any, 0, len(cols))
	for i, col := range cols {
		parts[i] = col + " LIKE ? ESCAPE '!'"
		args = append(args, pat)
	}
	return q.Where("("+strings.Join(parts, " OR ")+")", args...)
}

func applyUserKeyword(q *gorm.DB, kw string) *gorm.DB {
	kw = strings.TrimSpace(kw)
	if kw == "" {
		return q
	}
	pat := "%" + escapeLike(kw) + "%"
	like := " LIKE ? ESCAPE '!'"
	dict := " IN (SELECT value FROM dict_items WHERE type_code = ? AND label LIKE ? ESCAPE '!')"
	sql := strings.Join([]string{
		"users.username" + like,
		"users.nickname" + like,
		"users.email" + like,
		"users.phone" + like,
		"users.department" + like,
		"users.title" + like,
		"users.remark" + like,
		"users.last_login_ip" + like,
		"users.department" + dict,
		"users.gender" + dict,
		"users.status" + dict,
	}, " OR ")
	return q.Where("("+sql+")",
		pat, pat, pat, pat, pat, pat, pat, pat,
		seed.DictDepartment, pat,
		seed.DictGender, pat,
		seed.DictUserStatus, pat,
	)
}
