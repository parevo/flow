package engine

import (
	"sync/atomic"
)

type Telemetry struct {
	tasksProcessed uint64
	tasksFailed    uint64
	tasksRetried    uint64
}

var globalTelemetry = &Telemetry{}

func GetTelemetry() *Telemetry {
	return globalTelemetry
}

func (t *Telemetry) IncProcessed() {
	atomic.AddUint64(&t.tasksProcessed, 1)
}

func (t *Telemetry) IncFailed() {
	atomic.AddUint64(&t.tasksFailed, 1)
}

func (t *Telemetry) IncRetried() {
	atomic.AddUint64(&t.tasksRetried, 1)
}

func (t *Telemetry) GetStats() map[string]uint64 {
	return map[string]uint64{
		"processed": atomic.LoadUint64(&t.tasksProcessed),
		"failed":    atomic.LoadUint64(&t.tasksFailed),
		"retried":    atomic.LoadUint64(&t.tasksRetried),
	}
}
