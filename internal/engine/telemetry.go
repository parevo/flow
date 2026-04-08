package engine

import (
	"time"

	"github.com/parevo/flow/internal/telemetry"
)

// Telemetry bridge to Prometheus
type Telemetry struct{}

var globalTelemetry = &Telemetry{}

func GetTelemetry() *Telemetry {
	return globalTelemetry
}

func (t *Telemetry) IncProcessed(namespace, nodeType string, status string) {
	telemetry.StepsProcessed.WithLabelValues(namespace, nodeType, status).Inc()
}

func (t *Telemetry) ObserveDuration(namespace, nodeType string, duration time.Duration) {
	telemetry.StepDuration.WithLabelValues(namespace, nodeType).Observe(duration.Seconds())
}

func (t *Telemetry) WorkerStarted(id string) {
	telemetry.ActiveWorkers.WithLabelValues(id).Inc()
}

func (t *Telemetry) WorkerStopped(id string) {
	telemetry.ActiveWorkers.WithLabelValues(id).Dec()
}

// Keep legacy methods for basic worker support (mapped to default labels)
func (t *Telemetry) IncFailed()  { telemetry.StepsProcessed.WithLabelValues("unknown", "unknown", "failed").Inc() }
func (t *Telemetry) IncRetried() { telemetry.StepsProcessed.WithLabelValues("unknown", "unknown", "retried").Inc() }
func (t *Telemetry) IncProcessedLegacy() {
	telemetry.StepsProcessed.WithLabelValues("unknown", "unknown", "completed").Inc()
}
func (t *Telemetry) WorkerStartedLegacy() { telemetry.ActiveWorkers.WithLabelValues("unknown").Set(1) }
func (t *Telemetry) WorkerStoppedLegacy() { telemetry.ActiveWorkers.WithLabelValues("unknown").Set(0) }
