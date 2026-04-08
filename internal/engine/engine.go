package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/storage"
	"github.com/parevo/flow/internal/telemetry"
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

	telemetry.WorkflowsStarted.WithLabelValues(namespace, workflowID).Inc()

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

	// Mark non-matching branch targets as SKIPPED
	allEdges := g.Edges[step.NodeID]
	for _, edge := range allEdges {
		isTaken := false
		for _, n := range nextNodes {
			if n == edge.TargetID {
				isTaken = true
				break
			}
		}
		if !isTaken {
			e.skipNode(ctx, exec, g, edge.TargetID)
		}
	}

	if len(nextNodes) == 0 {
		return e.checkAndFinishExecution(ctx, exec)
	}

	for _, nextID := range nextNodes {
		ready, err := e.isNodeReady(ctx, nextID, exec.Namespace, exec.ID, g)
		if err != nil {
			return err
		}

		if ready {
			finalInput := output
			// 10/10 God-Tier: Check for JOIN and aggregate inputs
			if g.InDegree[nextID] > 1 {
				steps, _ := e.storage.GetExecutionSteps(ctx, exec.Namespace, exec.ID)
				aggMap := make(map[string]interface{})
				for _, pID := range g.Predecessors[nextID] {
					for _, s := range steps {
						if s.NodeID == pID {
							// In aggregation, use the latest output of each predecessor
							var outData interface{}
							_ = json.Unmarshal([]byte(s.Output), &outData)
							aggMap[pID] = outData
						}
					}
				}
				aggJSON, _ := json.Marshal(aggMap)
				finalInput = string(aggJSON)
			}

			newStep := &models.ExecutionStep{
				ID:          uuid.New().String(),
				Namespace:   exec.Namespace,
				ExecutionID: exec.ID,
				NodeID:      nextID,
				Status:      models.TaskPending,
				Input:       finalInput,
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

	// Merge signal data with existing input so downstream nodes (like condition) can access it
	mergedOutput := e.mergeSignalData(targetStep.Input, output)

	e.logger.Info("Signaling execution resume", "execution", execID, "node", nodeID)
	return e.CompleteStep(ctx, targetStep, mergedOutput)
}

// mergeSignalData combines existing input with signal data for downstream nodes
func (e *Engine) mergeSignalData(existingInput string, signalData string) string {
	// Try to parse signal data as JSON
	var signalMap map[string]interface{}
	signalErr := json.Unmarshal([]byte(signalData), &signalMap)

	// If signal data is not valid JSON or is empty, return it as-is for backwards compatibility
	if signalErr != nil || len(signalMap) == 0 {
		return signalData
	}

	// Parse existing input
	var inputMap map[string]interface{}
	if existingInput != "" {
		_ = json.Unmarshal([]byte(existingInput), &inputMap)
	}
	if inputMap == nil {
		inputMap = make(map[string]interface{})
	}


	// Merge: signal data overwrites existing keys
	for k, v := range signalMap {
		inputMap[k] = v
	}

	// Return merged JSON
	merged, _ := json.Marshal(inputMap)
	return string(merged)
}

func (e *Engine) skipNode(ctx context.Context, exec *models.Execution, g *Graph, nodeID string) {
	// 1. Check if already skipped/finished to avoid cycles or redundant work
	steps, _ := e.storage.GetExecutionSteps(ctx, exec.Namespace, exec.ID)
	for _, s := range steps {
		if s.NodeID == nodeID && (s.Status == models.TaskCompleted || s.Status == models.TaskSkipped || s.Status == models.TaskFailed) {
			return
		}
	}

	// 2. Mark as SKIPPED
	step := &models.ExecutionStep{
		ID:          uuid.New().String(),
		Namespace:   exec.Namespace,
		ExecutionID: exec.ID,
		NodeID:      nodeID,
		Status:      models.TaskSkipped,
		StartedAt:   time.Now(),
	}
	now := time.Now()
	step.FinishedAt = &now
	e.storage.CreateExecutionStep(ctx, exec.Namespace, step)
	e.logger.Debug("Skipping node", "node", nodeID, "execution", exec.ID)

	// 3. Propagate to children
	// A child should be skipped if ALL its predecessors are now SKIPPED.
	// If it has at least one COMPLETED predecessor, it might be ready to RUN instead.
	for _, edge := range g.Edges[nodeID] {
		childID := edge.TargetID

		ready, _ := e.isNodeReady(ctx, childID, exec.Namespace, exec.ID, g)
		if ready {
			// Check if all predecessors are skipped
			childSteps, _ := e.storage.GetExecutionSteps(ctx, exec.Namespace, exec.ID)
			allSkipped := true
			// Find all predecessors of childID
			preds := []string{}
			for src, edgs := range g.Edges {
				for _, edg := range edgs {
					if edg.TargetID == childID {
						preds = append(preds, src)
					}
				}
			}

			for _, p := range preds {
				found := false
				for _, cs := range childSteps {
					if cs.NodeID == p {
						if cs.Status != models.TaskSkipped {
							allSkipped = false
						}
						found = true
						break
					}
				}
				if !found || !allSkipped {
					allSkipped = false
					break
				}
			}

			if allSkipped {
				e.skipNode(ctx, exec, g, childID)
			} else {
				// At least one is COMPLETED! Start the node.
				newStep := &models.ExecutionStep{
					ID:          uuid.New().String(),
					Namespace:   exec.Namespace,
					ExecutionID: exec.ID,
					NodeID:      childID,
					Status:      models.TaskPending,
					Input:       "", // Or merge inputs?
					StartedAt:   time.Now(),
				}
				e.storage.CreateExecutionStep(ctx, exec.Namespace, newStep)
			}
		}
	}
}

// ReapTimeouts checks for WAITING steps that have passed their timeout period
func (e *Engine) ReapTimeouts(ctx context.Context, namespace string) error {
	// For simplicity in this version, we list all executions and their steps.
	// In production, we would use a more efficient query or a TTL index in Redis.
	execs, err := e.storage.ListExecutions(ctx, namespace)
	if err != nil {
		return err
	}

	for _, exec := range execs {
		if exec.Status != models.TaskRunning {
			continue
		}
		steps, err := e.storage.GetExecutionSteps(ctx, namespace, exec.ID)
		if err != nil {
			continue
		}
		for _, step := range steps {
			if step.Status == models.TaskWaiting {
				// Get node config to find timeout
				wf, err := e.storage.GetWorkflow(ctx, namespace, exec.WorkflowID)
				if err != nil {
					e.logger.Error("Failed to get workflow for reaper", "workflow", exec.WorkflowID, "error", err)
					continue
				}
				for _, n := range wf.Nodes {
					if n.ID == step.NodeID {
						if timeoutStr, ok := n.Config["timeout"].(string); ok {
							timeout, err := time.ParseDuration(timeoutStr)
							if err == nil {
								if time.Since(step.UpdatedAt) > timeout {
									e.logger.Warn("Step timed out waiting for signal", "step", step.ID, "node", step.NodeID)
									step.Status = models.TaskFailed
									step.Error = "SIGNAL_TIMEOUT"
									now := time.Now()
									step.FinishedAt = &now
									e.storage.UpdateExecutionStep(ctx, namespace, step)
									e.checkAndFinishExecution(ctx, exec)
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (e *Engine) GetExecutionStatus(ctx context.Context, namespace string, execID string) (*models.Execution, error) {
	return e.storage.GetExecution(ctx, namespace, execID)
}

func (e *Engine) GetLogger() *slog.Logger {
	return e.logger
}

// RegisterWorkflow saves a workflow definition to storage
func (e *Engine) RegisterWorkflow(ctx context.Context, namespace string, wf *models.Workflow) error {
	wf.Namespace = namespace
	wf.Status = models.WorkflowActive
	wf.CreatedAt = time.Now()
	wf.UpdatedAt = time.Now()
	return e.storage.SaveWorkflow(ctx, namespace, wf)
}

// Execute is an alias for StartWorkflow (accepts []byte input)
func (e *Engine) Execute(ctx context.Context, namespace, workflowID string, input []byte) (string, error) {
	return e.StartWorkflow(ctx, namespace, workflowID, string(input))
}

// StartWorker starts a worker loop that processes tasks
func (e *Engine) StartWorker(ctx context.Context, namespace, workerID string) {
	worker := NewWorker(workerID, e, e.registry, 100*time.Millisecond)
	worker.SetNamespace(namespace)
	worker.Start(ctx)
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
		if s.Status == models.TaskCompleted || s.Status == models.TaskSkipped {
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
			e.logger.Info("Execution failed", "execution", exec.ID, "error", firstError)
			telemetry.WorkflowsFailed.WithLabelValues(exec.Namespace, exec.WorkflowID, firstError).Inc()
		} else {
			exec.Status = models.TaskCompleted
			exec.Output = finalOutput
			e.logger.Info("Execution completed", "execution", exec.ID)
			telemetry.WorkflowsCompleted.WithLabelValues(exec.Namespace, exec.WorkflowID).Inc()
		}
		now := time.Now()
		exec.FinishedAt = &now
		return e.storage.UpdateExecution(ctx, exec.Namespace, exec)
	}

	return nil
}
