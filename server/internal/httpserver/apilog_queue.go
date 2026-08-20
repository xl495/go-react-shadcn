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
	ch      chan models.APILog
	enabled bool
	sampleN uint64
	counter atomic.Uint64
	stop    chan struct{}
	done    chan struct{}
	flushMu sync.Mutex
}

func newAPILogQueue(db *gorm.DB, enabled bool, sampleN int) *apiLogQueue {
	if sampleN < 1 {
		sampleN = 1
	}
	q := &apiLogQueue{
		db:      db,
		ch:      make(chan models.APILog, 256),
		enabled: enabled,
		sampleN: uint64(sampleN),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	if enabled {
		go q.loop()
	} else {
		close(q.done)
	}
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
	case q.ch <- row:
	default:
		// drop under backpressure
	}
}

func (q *apiLogQueue) loop() {
	defer close(q.done)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	buf := make([]models.APILog, 0, 32)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		q.flushMu.Lock()
		_ = q.db.Create(&buf).Error
		q.flushMu.Unlock()
		buf = buf[:0]
	}
	for {
		select {
		case row := <-q.ch:
			buf = append(buf, row)
			if len(buf) >= 32 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-q.stop:
			for {
				select {
				case row := <-q.ch:
					buf = append(buf, row)
				default:
					flush()
					return
				}
			}
		}
	}
}

func (q *apiLogQueue) Flush() {
	if q == nil || !q.enabled {
		return
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(q.ch) == 0 {
			time.Sleep(250 * time.Millisecond)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (q *apiLogQueue) Stop() {
	if q == nil || !q.enabled {
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
