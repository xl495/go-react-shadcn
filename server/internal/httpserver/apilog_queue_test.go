package httpserver

import (
	"testing"

	"go-react-shadcn/internal/models"
)

func TestAuditQueueOverflowSyncs(t *testing.T) {
	app := testApp(t)
	q := app.apiLogs
	q.Stop()
	q.enabled = true
	q.sampleN = 1
	for i := 0; i < cap(q.api); i++ {
		select {
		case q.api <- models.APILog{TraceID: "fill", Method: "GET", Path: "/fill"}:
		default:
		}
	}
	q.enqueue(models.APILog{TraceID: "overflow", Method: "POST", Path: "/overflow"})
	if q.droppedCount() != 1 {
		t.Fatalf("dropped=%d", q.droppedCount())
	}
	var n int64
	if err := app.DB.Model(&models.APILog{}).Where("path = ?", "/overflow").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("sync write n=%d", n)
	}
}
