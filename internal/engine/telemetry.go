package engine

import (
	"fmt"
	"sync/atomic"
)

// Zero-Dependency Telemetry using standard sync/atomic
type Telemetry struct {
	tasksProcessed uint64
	tasksFailed    uint64
	tasksRetried    uint64
	activeWorkers  int64
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

func (t *Telemetry) WorkerStarted() {
	atomic.AddInt64(&t.activeWorkers, 1)
}

func (t *Telemetry) WorkerStopped() {
	atomic.AddInt64(&t.activeWorkers, -1)
}

// ToPrometheusFormat generates the metric string in Prometheus text format manually.
// This allows us to avoid a heavy dependency on the Prometheus client library.
func (t *Telemetry) ToPrometheusFormat() string {
	return fmt.Sprintf(
		"# HELP parevo_flow_tasks_processed_total The total number of processed tasks\n"+
			"# TYPE parevo_flow_tasks_processed_total counter\n"+
			"parevo_flow_tasks_processed_total %d\n"+
			"# HELP parevo_flow_tasks_failed_total The total number of failed tasks\n"+
			"# TYPE parevo_flow_tasks_failed_total counter\n"+
			"parevo_flow_tasks_failed_total %d\n"+
			"# HELP parevo_flow_tasks_retried_total The total number of retried tasks\n"+
			"# TYPE parevo_flow_tasks_retried_total counter\n"+
			"parevo_flow_tasks_retried_total %d\n"+
			"# HELP parevo_flow_active_workers The number of currently active workers\n"+
			"# TYPE parevo_flow_active_workers gauge\n"+
			"parevo_flow_active_workers %d\n",
		atomic.LoadUint64(&t.tasksProcessed),
		atomic.LoadUint64(&t.tasksFailed),
		atomic.LoadUint64(&t.tasksRetried),
		atomic.LoadInt64(&t.activeWorkers),
	)
}

func (t *Telemetry) GetStats() map[string]float64 {
	return map[string]float64{
		"processed": float64(atomic.LoadUint64(&t.tasksProcessed)),
		"failed":    float64(atomic.LoadUint64(&t.tasksFailed)),
		"retried":   float64(atomic.LoadUint64(&t.tasksRetried)),
		"workers":   float64(atomic.LoadInt64(&t.activeWorkers)),
	}
}
