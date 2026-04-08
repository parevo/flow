package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WorkflowsStarted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "flow_workflows_started_total",
		Help: "Total number of workflows started",
	}, []string{"namespace", "workflow_id"})

	WorkflowsCompleted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "flow_workflows_completed_total",
		Help: "Total number of workflows completed successfully",
	}, []string{"namespace", "workflow_id"})

	WorkflowsFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "flow_workflows_failed_total",
		Help: "Total number of workflows failed",
	}, []string{"namespace", "workflow_id", "error"})

	StepsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "flow_steps_processed_total",
		Help: "Total number of steps processed by workers",
	}, []string{"namespace", "node_type", "status"})

	ActiveWorkers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "flow_active_workers",
		Help: "Number of currently active workers",
	}, []string{"worker_id"})

	StepDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "flow_step_duration_seconds",
		Help: "Duration of step execution",
		Buckets: prometheus.DefBuckets,
	}, []string{"namespace", "node_type"})
)
