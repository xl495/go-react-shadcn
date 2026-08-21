package httpserver

import (
	"sync"
	"sync/atomic"
	"time"

	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

type apiLogQueue struct {
	db      *gorm.DB
	api     chan models.APILog
	op      chan models.OpLog
	login   chan models.LoginLog
	flushCh chan chan struct{}
	enabled bool
	sampleN uint64 // set from a clamped positive int sample rate
	counter atomic.Uint64
	dropped atomic.Uint64
	stop    chan struct{}
	done    chan struct{}
	flushMu sync.Mutex
}

func newAPILogQueue(db *gorm.DB, enabled bool, sampleN int) *apiLogQueue {
	n := uint64(1)
	if sampleN > 1 {
		n = uint64(sampleN) //nolint:gosec // sample rate comes from config, always a small positive int
	}
	q := &apiLogQueue{
		db:      db,
		api:     make(chan models.APILog, 256),
		op:      make(chan models.OpLog, 256),
		login:   make(chan models.LoginLog, 256),
		flushCh: make(chan chan struct{}, 8),
		enabled: enabled,
		sampleN: n,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go q.loop()
	return q
}

func (q *apiLogQueue) enqueue(row models.APILog) {
	if q == nil || !q.enabled {
		return
	}
	n := q.counter.Add(1)
	if q.sampleN > 1 && n%q.sampleN != 0 {
		return
	}
	select {
	case q.api <- row:
	default:
		q.dropped.Add(1)
		_ = q.db.Create(&row).Error
	}
}

func (q *apiLogQueue) enqueueOp(row models.OpLog) {
	if q == nil {
		return
	}
	select {
	case q.op <- row:
	default:
		q.dropped.Add(1)
		_ = q.db.Create(&row).Error
	}
}

func (q *apiLogQueue) enqueueLogin(row models.LoginLog) {
	if q == nil {
		return
	}
	select {
	case q.login <- row:
	default:
		q.dropped.Add(1)
		_ = q.db.Create(&row).Error
	}
}

func (q *apiLogQueue) loop() {
	defer close(q.done)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	apis := make([]models.APILog, 0, 32)
	ops := make([]models.OpLog, 0, 32)
	logins := make([]models.LoginLog, 0, 32)
	flush := func() {
		q.flushMu.Lock()
		defer q.flushMu.Unlock()
		if len(apis) > 0 {
			_ = q.db.Create(&apis).Error
			apis = apis[:0]
		}
		if len(ops) > 0 {
			_ = q.db.Create(&ops).Error
			ops = ops[:0]
		}
		if len(logins) > 0 {
			_ = q.db.Create(&logins).Error
			logins = logins[:0]
		}
	}
	drain := func() {
		for {
			select {
			case row := <-q.api:
				apis = append(apis, row)
			case row := <-q.op:
				ops = append(ops, row)
			case row := <-q.login:
				logins = append(logins, row)
			default:
				flush()
				return
			}
		}
	}
	for {
		select {
		case row := <-q.api:
			apis = append(apis, row)
			if len(apis) >= 32 {
				flush()
			}
		case row := <-q.op:
			ops = append(ops, row)
			if len(ops) >= 32 {
				flush()
			}
		case row := <-q.login:
			logins = append(logins, row)
			if len(logins) >= 32 {
				flush()
			}
		case <-ticker.C:
			flush()
		case done := <-q.flushCh:
			drain()
			close(done)
		case <-q.stop:
			drain()
			return
		}
	}
}

func (q *apiLogQueue) droppedCount() uint64 {
	if q == nil {
		return 0
	}
	return q.dropped.Load()
}

func (q *apiLogQueue) Flush() {
	if q == nil {
		return
	}
	done := make(chan struct{})
	select {
	case <-q.stop:
		return
	case q.flushCh <- done:
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	case <-time.After(2 * time.Second):
	}
}

func (q *apiLogQueue) Stop() {
	if q == nil {
		return
	}
	select {
	case <-q.stop:
		return
	default:
		close(q.stop)
	}
	<-q.done
}
