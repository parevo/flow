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
	GetTelemetry().WorkerStarted()
	defer GetTelemetry().WorkerStopped()
	
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
		w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step)
		return
	}

	GetTelemetry().IncProcessed()
	w.engine.logger.Info("Processing step", "worker", w.id, "step", step.ID, "node", step.NodeID, "namespace", step.Namespace)

	output, err := w.executeStep(ctx, step)
	if err != nil {
		w.engine.logger.Error("Step execution failed", "worker", w.id, "step", step.ID, "attempt", step.AttemptNumber, "error", err)
		
		maxRetries := 5
		if step.AttemptNumber < maxRetries {
			step.AttemptNumber++
			delay := time.Duration(1<<uint(step.AttemptNumber)) * 10 * time.Second
			nextScheduled := time.Now().Add(delay)
			step.ScheduledAt = &nextScheduled
			step.Status = models.TaskPending
			step.Error = err.Error()
			
			w.engine.logger.Info("Rescheduling step with backoff", "worker", w.id, "step", step.ID, "next_run", nextScheduled.Format(time.RFC3339))
			GetTelemetry().IncRetried()
			w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step)
			return
		}

		step.Status = models.TaskFailed
		step.Error = fmt.Sprintf("Max retries reached: %v", err)
		now := time.Now()
		step.FinishedAt = &now
		GetTelemetry().IncFailed()
		w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step)
		return
	}

	if err := w.engine.CompleteStep(ctx, step, output); err != nil {
		w.engine.logger.Error("Failed to complete step", "worker", w.id, "step", step.ID, "error", err)
	}
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
