package httpserver

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func writeCSV(c *gin.Context, filename string, header []string, rows [][]string) {
	w := beginCSV(c, filename, header)
	_ = w.WriteAll(rows)
}

func beginCSV(c *gin.Context, filename string, header []string) *csv.Writer {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Status(http.StatusOK)
	w := csv.NewWriter(c.Writer)
	_ = w.Write(header)
	return w
}

func streamCSV[T any](c *gin.Context, db *gorm.DB, q *gorm.DB, filename string, header []string, mapRow func(T) []string) error {
	rows, err := q.Limit(5000).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	w := beginCSV(c, filename, header)
	defer w.Flush()
	for rows.Next() {
		var row T
		if err := db.ScanRows(rows, &row); err != nil {
			return err
		}
		if err := w.Write(mapRow(row)); err != nil {
			return err
		}
	}
	return rows.Err()
}

func formatUint(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}
