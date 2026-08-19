package rbac

import (
	"fmt"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

const modelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := newAdapter(db)
	if err != nil {
		return nil, fmt.Errorf("casbin adapter: %w", err)
	}
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("casbin model: %w", err)
	}
	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("casbin enforcer: %w", err)
	}
	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("casbin load: %w", err)
	}
	return e, nil
}

import (
	"fmt"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

const modelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := newAdapter(db)
	if err != nil {
		return nil, fmt.Errorf("casbin adapter: %w", err)
	}
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("casbin model: %w", err)
	}
	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("casbin enforcer: %w", err)
	}
	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("casbin load: %w", err)
	}
	return e, nil
}

func NewPoolEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := newAdapter(db)
	if err != nil {
		return nil, fmt.Errorf("casbin adapter: %w", err)
	}
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("casbin model: %w", err)
	}
	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("casbin enforcer: %w", err)
	}
	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("casbin load: %w", err)
	}

	// Query optimization: add simple in-memory query cache for common patterns
	cache := &sync.Map{}
	e.SetAdapter(adapter) // ensure adapter is set

	var pool *sync.Pool
	pool = &sync.Pool{
		New: func() any {
			e2, _ := casbin.NewEnforcer(m, adapter)
			e2.LoadPolicy()
			return e2
		},
	}

	// eviction via ticker + cleanup
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			// occasionally reset the pool to force recreation (clears any cached state)
			pool = &sync.Pool{
				New: func() any {
					e2, _ := casbin.NewEnforcer(m, adapter)
					e2.LoadPolicy()
					return e2
				},
			}
		}
	}()

	return e, nil
}
