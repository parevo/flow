package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/parevo/flow/internal/models"
)

type Worker struct {
	id        string
	namespace string // Optional: filter by namespace
	engine    *Engine
	registry  *Registry
	interval  time.Duration
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
	log.Printf("Worker %s started (Namespace: %s)\n", w.id, w.namespace)
	GetTelemetry().WorkerStarted()
	defer GetTelemetry().WorkerStopped()
	
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s stopping\n", w.id)
			return
		case <-ticker.C:
			w.processOne(ctx)
		}
	}
}

func (w *Worker) processOne(ctx context.Context) {
	// ClaimReadyStep now takes namespace
	step, err := w.engine.storage.ClaimReadyStep(ctx, w.namespace, w.id)
	if err != nil {
		log.Printf("Worker %s: failed to claim step: %v\n", w.id, err)
		return
	}

	if step == nil {
		return // No work
	}

	GetTelemetry().IncProcessed()
	log.Printf("Worker %s: processing step %s (Namespace: %s, Node: %s)\n", w.id, step.ID, step.Namespace, step.NodeID)

	// Execute node logic
	output, err := w.executeStep(ctx, step)
	if err != nil {
		log.Printf("Worker %s: step %s failed (Attempt %d): %v\n", w.id, step.ID, step.AttemptNumber, err)
		
		// HIGH-LOAD: Handle retries with Exponential Backoff
		maxRetries := 5
		if step.AttemptNumber < maxRetries {
			step.AttemptNumber++
			
			// Exponential Backoff: 2^attempt * 10 seconds
			delay := time.Duration(1<<uint(step.AttemptNumber)) * 10 * time.Second
			nextScheduled := time.Now().Add(delay)
			step.ScheduledAt = &nextScheduled
			step.Status = models.TaskPending
			step.Error = err.Error()
			
			log.Printf("Worker %s: rescheduling step %s for %v\n", w.id, step.ID, nextScheduled.Format(time.RFC3339))
			GetTelemetry().IncRetried()
			w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step)
			return
		}

		// Hard Fail after max retries
		step.Status = models.TaskFailed
		step.Error = fmt.Sprintf("Max retries reached: %v", err)
		now := time.Now()
		step.FinishedAt = &now
		GetTelemetry().IncFailed()
		w.engine.storage.UpdateExecutionStep(ctx, step.Namespace, step)
		return
	}

	// Complete step and trigger transitions
	if err := w.engine.CompleteStep(ctx, step, output); err != nil {
		log.Printf("Worker %s: failed to complete step %s: %v\n", w.id, step.ID, err)
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
