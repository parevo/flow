package engine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	tasksProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "parevo_flow_tasks_processed_total",
		Help: "The total number of processed tasks",
	})
	tasksFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "parevo_flow_tasks_failed_total",
		Help: "The total number of failed tasks",
	})
	tasksRetried = promauto.NewCounter(prometheus.CounterOpts{
		Name: "parevo_flow_tasks_retried_total",
		Help: "The total number of retried tasks",
	})
	activeWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "parevo_flow_active_workers",
		Help: "The number of currently active workers",
	})
)

type Telemetry struct{}

var globalTelemetry = &Telemetry{}

func GetTelemetry() *Telemetry {
	return globalTelemetry
}

func (t *Telemetry) IncProcessed() {
	tasksProcessed.Inc()
}

func (t *Telemetry) IncFailed() {
	tasksFailed.Inc()
}

func (t *Telemetry) IncRetried() {
	tasksRetried.Inc()
}

func (t *Telemetry) WorkerStarted() {
	activeWorkers.Inc()
}

func (t *Telemetry) WorkerStopped() {
	activeWorkers.Dec()
}

func (t *Telemetry) GetStats() map[string]float64 {
	// Note: In production, you'd use the Prometheus registry to get these.
	// This helper exists for backward compatibility with our log-based metrics.
	return map[string]float64{
		"processed": 0, // Placeholder
		"failed":    0,
		"retried":   0,
	}
}
