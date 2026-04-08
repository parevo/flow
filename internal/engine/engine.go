package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/storage"
)

type Engine struct {
	storage  storage.Storage
	registry *Registry
	logger   *slog.Logger
}

func NewEngine(s storage.Storage, r *Registry) *Engine {
	return &Engine{
		storage:  s,
		registry: r,
		logger:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (e *Engine) WithLogger(l *slog.Logger) *Engine {
	e.logger = l
	return e
}

// StartWorkflow creates a new execution for a workflow
func (e *Engine) StartWorkflow(ctx context.Context, namespace, workflowID string, input string) (string, error) {
	wf, err := e.storage.GetWorkflow(ctx, namespace, workflowID)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow: %w", err)
	}

	execID := uuid.New().String()
	exec := &models.Execution{
		ID:         execID,
		Namespace:  namespace,
		WorkflowID: workflowID,
		Version:    wf.Version,
		Status:     models.TaskRunning,
		Input:      input,
		StartedAt:  time.Now(),
	}

	if err := e.storage.CreateExecution(ctx, namespace, exec); err != nil {
		return "", fmt.Errorf("failed to create execution: %w", err)
	}

	// Build graph to find initial nodes
	g, err := NewGraph(wf)
	if err != nil {
		return "", err
	}

	// Early Validation: Check all nodes in the graph
	for _, n := range wf.Nodes {
		executor, err := e.registry.Get(n.Type)
		if err != nil {
			return "", fmt.Errorf("node %s: %w", n.ID, err)
		}
		if err := executor.Validate(n.Config); err != nil {
			return "", fmt.Errorf("node %s configuration error: %w", n.ID, err)
		}
	}

	initialNodeIDs := g.GetInitialNodes()
	for _, nodeID := range initialNodeIDs {
		step := &models.ExecutionStep{
			ID:          uuid.New().String(),
			Namespace:   namespace,
			ExecutionID: execID,
			NodeID:      nodeID,
			Status:      models.TaskPending,
			Input:       input, // First nodes get the execution input
			StartedAt:   time.Now(),
		}
		if err := e.storage.CreateExecutionStep(ctx, namespace, step); err != nil {
			return "", err
		}
	}

	return execID, nil
}

// CompleteStep handles the transition after a node finishes
func (e *Engine) CompleteStep(ctx context.Context, step *models.ExecutionStep, output string) error {
	step.Status = models.TaskCompleted
	step.Output = output
	now := time.Now()
	step.FinishedAt = &now

	if err := e.storage.UpdateExecutionStep(ctx, step.Namespace, step); err != nil {
		return err
	}

	// Find the execution to get the workflow
	exec, err := e.storage.GetExecution(ctx, step.Namespace, step.ExecutionID)
	if err != nil {
		return err
	}

	wf, err := e.storage.GetWorkflow(ctx, step.Namespace, exec.WorkflowID)
	if err != nil {
		return err
	}

	g, err := NewGraph(wf)
	if err != nil {
		return err
	}

	// Find next nodes with branch-aware routing
	nextNodes := g.GetNextNodesWithBranch(step.NodeID, output)
	if len(nextNodes) == 0 {
		return e.checkAndFinishExecution(ctx, exec)
	}

	for _, nextID := range nextNodes {
		ready, err := e.isNodeReady(ctx, nextID, exec.Namespace, exec.ID, g)
		if err != nil {
			return err
		}

		if ready {
			newStep := &models.ExecutionStep{
				ID:          uuid.New().String(),
				Namespace:   exec.Namespace,
				ExecutionID: step.ExecutionID,
				NodeID:      nextID,
				Status:      models.TaskPending,
				Input:       output,
				StartedAt:   time.Now(),
			}
			if err := e.storage.CreateExecutionStep(ctx, exec.Namespace, newStep); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Engine) CancelExecution(ctx context.Context, namespace string, execID string) error {
	exec, err := e.storage.GetExecution(ctx, namespace, execID)
	if err != nil {
		return err
	}
	if exec.Status == models.TaskCompleted || exec.Status == models.TaskFailed {
		return fmt.Errorf("cannot cancel finished execution")
	}
	exec.Status = models.TaskCancelled
	now := time.Now()
	exec.FinishedAt = &now
	return e.storage.UpdateExecution(ctx, namespace, exec)
}

func (e *Engine) FailExecution(ctx context.Context, namespace string, execID string, message string) error {
	exec, err := e.storage.GetExecution(ctx, namespace, execID)
	if err != nil {
		return err
	}
	exec.Status = models.TaskFailed
	exec.ErrorMessage = message
	now := time.Now()
	exec.FinishedAt = &now
	return e.storage.UpdateExecution(ctx, namespace, exec)
}

func (e *Engine) SignalExecution(ctx context.Context, namespace string, execID string, nodeID string, output string) error {
	steps, err := e.storage.GetExecutionSteps(ctx, namespace, execID)
	if err != nil {
		return err
	}

	var targetStep *models.ExecutionStep
	for _, s := range steps {
		if s.NodeID == nodeID && s.Status == models.TaskWaiting {
			targetStep = s
			break
		}
	}

	if targetStep == nil {
		return fmt.Errorf("no WAITING step found for node %s in execution %s", nodeID, execID)
	}

	e.logger.Info("Signaling execution resume", "execution", execID, "node", nodeID)
	return e.CompleteStep(ctx, targetStep, output)
}

func (e *Engine) GetExecutionStatus(ctx context.Context, namespace string, execID string) (*models.Execution, error) {
	return e.storage.GetExecution(ctx, namespace, execID)
}

func (e *Engine) GetLogger() *slog.Logger {
	return e.logger
}

func (e *Engine) GetExecutionSteps(ctx context.Context, namespace string, execID string) ([]*models.ExecutionStep, error) {
	return e.storage.GetExecutionSteps(ctx, namespace, execID)
}

func (e *Engine) isNodeReady(ctx context.Context, nodeID string, namespace string, execID string, g *Graph) (bool, error) {
	predecessors := []string{}
	for source, edges := range g.Edges {
		for _, e := range edges {
			if e.TargetID == nodeID {
				predecessors = append(predecessors, source)
			}
		}
	}

	if len(predecessors) == 0 {
		return true, nil
	}

	steps, err := e.storage.GetExecutionSteps(ctx, namespace, execID)
	if err != nil {
		return false, err
	}

	completedNodes := make(map[string]bool)
	for _, s := range steps {
		if s.Status == models.TaskCompleted {
			completedNodes[s.NodeID] = true
		}
	}

	for _, p := range predecessors {
		if !completedNodes[p] {
			return false, nil
		}
	}

	return true, nil
}

func (e *Engine) checkAndFinishExecution(ctx context.Context, exec *models.Execution) error {
	steps, err := e.storage.GetExecutionSteps(ctx, exec.Namespace, exec.ID)
	if err != nil {
		return err
	}

	allFinished := true
	hasFailure := false
	var finalOutput string
	var firstError string

	for _, s := range steps {
		if s.Status == models.TaskFailed {
			hasFailure = true
			firstError = s.Error
		}
		if s.Status != models.TaskCompleted && s.Status != models.TaskFailed && s.Status != models.TaskSkipped {
			allFinished = false
			break
		}
		finalOutput = s.Output
	}

	if allFinished {
		if hasFailure {
			exec.Status = models.TaskFailed
			exec.ErrorMessage = firstError
		} else {
			exec.Status = models.TaskCompleted
			exec.Output = finalOutput
		}
		now := time.Now()
		exec.FinishedAt = &now
		return e.storage.UpdateExecution(ctx, exec.Namespace, exec)
	}

	return nil
}
