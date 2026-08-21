package httpserver

import (
	"sync"

	"go-react-shadcn/internal/models"
)

type deptCache struct {
	mu   sync.Mutex
	rows []models.Department
	ok   bool
}

func (d *deptCache) get() ([]models.Department, bool) {
	if d == nil {
		return nil, false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.ok {
		return nil, false
	}
	out := make([]models.Department, len(d.rows))
	copy(out, d.rows)
	return out, true
}

func (d *deptCache) put(rows []models.Department) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rows = append([]models.Department(nil), rows...)
	d.ok = true
}

func (d *deptCache) invalidate() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rows = nil
	d.ok = false
}
