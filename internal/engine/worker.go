package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/parevo/flow/internal/models"
)

type Worker struct {
	id        string
	namespace string
	engine    *Engine
	registry  *Registry
	interval  time.Duration
	mu        sync.Mutex
	running   bool
}

func NewWorker(id string, engine *Engine, registry *Registry, interval time.Duration) *Worker {
	return &Worker{
		id:       id,
		engine:   engine,
		registry: registry,
		interval: interval,
	}
}

// SetNamespace pins the worker to a specific namespace
func (w *Worker) SetNamespace(namespace string) {
	w.namespace = namespace
}

func (w *Worker) Start(ctx context.Context) {
	w.engine.logger.Info("Worker started", "id", w.id, "namespace", w.namespace)
	GetTelemetry().WorkerStarted(w.id)
	defer GetTelemetry().WorkerStopped(w.id)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.engine.logger.Info("Worker received shutdown signal", "id", w.id)
			return
		case <-ticker.C:
			w.processOne(ctx)
		}
	}
}

func (w *Worker) processOne(ctx context.Context) {
	w.mu.Lock()
	w.running = true
	defer func() {
		w.mu.Unlock()
		w.running = false
	}()

	step, err := w.engine.storage.ClaimReadyStep(ctx, w.namespace, w.id)
	if err != nil {
		w.engine.logger.Error("Failed to claim step", "id", w.id, "error", err)
		return
	}

	if step == nil {
		return
	}

	exec, err := w.engine.storage.GetExecution(ctx, step.Namespace, step.ExecutionID)
	if err == nil && exec.Status == models.TaskCancelled {
		w.engine.logger.Warn("Step skipped due to cancellation", "worker", w.id, "step", step.ID, "execution", exec.ID)
		step.Status = models.TaskCancelled
		now := time.Now()
		step.FinishedAt = &now
		if err := w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step); err != nil {
			w.engine.logger.Error("Failed to update execution step for cancellation", "worker", w.id, "step", step.ID, "error", err)
		}
		return
	}

	// We'll increment processed after finding node type below
	w.engine.logger.Info("Processing step", "worker", w.id, "step", step.ID, "node", step.NodeID, "namespace", step.Namespace)

	start := time.Now()
	output, err := w.executeStep(ctx, step)
	duration := time.Since(start)

	// Determine node type for metrics
	nodeType := "unknown"
	wf, _ := w.engine.storage.GetWorkflow(ctx, step.Namespace, exec.WorkflowID)
	for _, n := range wf.Nodes {
		if n.ID == step.NodeID {
			nodeType = n.Type
			break
		}
	}

	if err != nil {
		// Native Signal Support: Check for SIGNAL_WAIT
		if err.Error() == "SIGNAL_WAIT" {
			w.engine.logger.Info("Step entering WAITING state for external signal", "worker", w.id, "step", step.ID, "node", step.NodeID)
			step.Status = models.TaskWaiting
			if err := w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step); err != nil {
				w.engine.logger.Error("Failed to update execution step to WAITING state", "worker", w.id, "step", step.ID, "error", err)
			}
			GetTelemetry().IncProcessed(step.Namespace, nodeType, "waiting")
			return
		}

		w.engine.logger.Error("Step execution failed", "worker", w.id, "step", step.ID, "attempt", step.AttemptNumber, "error", err)

		// Dynamic Retry Policy
		policy := w.getRetryPolicy(wf, step.NodeID)

		if step.AttemptNumber < policy.MaxAttempts {
			step.AttemptNumber++
			delay := w.calculateBackoff(policy, step.AttemptNumber)

			nextScheduled := time.Now().Add(delay)
			step.ScheduledAt = &nextScheduled
			step.Status = models.TaskPending
			step.Error = err.Error()

			w.engine.logger.Info("Rescheduling step with dynamic backoff", "worker", w.id, "step", step.ID, "next_run", nextScheduled.Format(time.RFC3339))
			GetTelemetry().IncProcessed(step.Namespace, nodeType, "retried")
			if err := w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step); err != nil {
				w.engine.logger.Error("Failed to update execution step for retry", "worker", w.id, "step", step.ID, "error", err)
			}
			return
		}

		// Hard Fail
		step.Status = models.TaskFailed
		step.Error = fmt.Sprintf("Max retries reached: %v", err)
		now := time.Now()
		step.FinishedAt = &now
		GetTelemetry().IncProcessed(step.Namespace, nodeType, "failed")
		GetTelemetry().ObserveDuration(step.Namespace, nodeType, duration)
		if err := w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step); err != nil {
			w.engine.logger.Error("Failed to update execution step after max retries", "worker", w.id, "step", step.ID, "error", err)
		}

		// Check if compensation exists - if yes, don't fail execution yet
		hasCompensation := w.handleCompensation(ctx, wf, step, exec, err)
		if !hasCompensation {
			if err := w.engine.checkAndFinishExecution(ctx, exec); err != nil {
				w.engine.logger.Error("Failed to check and finish execution", "worker", w.id, "execution", exec.ID, "error", err)
			}
		}
		return
	}

	GetTelemetry().IncProcessed(step.Namespace, nodeType, "completed")
	GetTelemetry().ObserveDuration(step.Namespace, nodeType, duration)
	if err := w.engine.CompleteStep(ctx, step, output); err != nil {
		w.engine.logger.Error("Failed to complete step", "worker", w.id, "step", step.ID, "error", err)
	}
}

func (w *Worker) getRetryPolicy(wf *models.Workflow, nodeID string) *models.RetryPolicy {
	for _, n := range wf.Nodes {
		if n.ID == nodeID {
			if n.RetryPolicy != nil {
				return n.RetryPolicy
			}
			break
		}
	}
	return &models.RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    5 * time.Minute,
	}
}

func (w *Worker) calculateBackoff(policy *models.RetryPolicy, attempt int) time.Duration {
	delay := time.Duration(float64(policy.InitialInterval) * (policy.BackoffCoefficient))
	if attempt > 1 {
		delay = time.Duration(float64(policy.InitialInterval) * float64(attempt) * policy.BackoffCoefficient)
	}
	if delay > policy.MaximumInterval {
		delay = policy.MaximumInterval
	}
	return delay
}

func (w *Worker) handleCompensation(ctx context.Context, wf *models.Workflow, step *models.ExecutionStep, exec *models.Execution, err error) bool {
	for _, n := range wf.Nodes {
		if n.ID == step.NodeID && n.CompensateNodeID != "" {
			w.engine.logger.Info("Triggering Saga Compensation", "execution", exec.ID, "failed_node", n.ID, "compensation_node", n.CompensateNodeID)
			compStep := &models.ExecutionStep{
				ID:          fmt.Sprintf("comp-%s", step.ID),
				Namespace:   step.Namespace,
				ExecutionID: step.ExecutionID,
				NodeID:      n.CompensateNodeID,
				Status:      models.TaskPending,
				Input:       fmt.Sprintf(`{"reason": "compensation", "failedNode": "%s", "error": "%s"}`, n.ID, err.Error()),
				StartedAt:   time.Now(),
			}
			if err := w.engine.storage.CreateExecutionStep(ctx, step.Namespace, compStep); err != nil {
				w.engine.logger.Error("Failed to create compensation step", "execution", exec.ID, "step", compStep.ID, "error", err)
			}
			return true // Compensation exists, don't fail execution yet
		}
	}
	return false // No compensation found
}

func (w *Worker) executeStep(ctx context.Context, step *models.ExecutionStep) (string, error) {
	// Find the execution to get the node config from workflow (In real apps, use a cache)
	exec, err := w.engine.storage.GetExecution(ctx, step.Namespace, step.ExecutionID)
	if err != nil {
		return "", err
	}
	wf, err := w.engine.storage.GetWorkflow(ctx, step.Namespace, exec.WorkflowID)
	if err != nil {
		return "", err
	}

	var nodeConfig map[string]interface{}
	for _, n := range wf.Nodes {
		if n.ID == step.NodeID {
			nodeConfig = n.Config
			executor, err := w.registry.Get(n.Type)
			if err != nil {
				return "", err
			}
			return executor.Execute(ctx, nodeConfig, step.Input)
		}
	}

	return "", fmt.Errorf("node %s not found in workflow", step.NodeID)
}
