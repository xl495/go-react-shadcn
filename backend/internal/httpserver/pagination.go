package httpserver

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type pageParams struct {
	Page     int
	PageSize int
	Sort     string
	Order    string
}

type pageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func parsePage(c *gin.Context, defaultSize, maxSize int) pageParams {
	page := 1
	size := defaultSize
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= maxSize {
			size = n
		}
	}
	sort := c.DefaultQuery("sort", "id")
	order := c.DefaultQuery("order", "asc")
	if order != "desc" {
		order = "asc"
	}
	return pageParams{Page: page, PageSize: size, Sort: sort, Order: order}
}

func (p pageParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p pageParams) OrderClause(allowed map[string]string, fallback string) string {
	col, ok := allowed[p.Sort]
	if !ok {
		col = fallback
	}
	if p.Order == "desc" {
		return col + " desc"
	}
	return col + " asc"
}
