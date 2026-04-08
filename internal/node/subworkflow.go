package node

import (
	"context"
	"fmt"
	"time"

	"github.com/parevo/flow/internal/models"
)

// SubWorkflowEngine interface to avoid circular dependency
type SubWorkflowEngine interface {
	StartWorkflow(ctx context.Context, namespace, workflowID string, input string) (string, error)
	GetExecutionStatus(ctx context.Context, namespace, execID string) (*models.Execution, error)
}

type SubWorkflowNode struct {
	engine SubWorkflowEngine
}

func NewSubWorkflowNode(e SubWorkflowEngine) *SubWorkflowNode {
	return &SubWorkflowNode{engine: e}
}

func (s *SubWorkflowNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	childWFID, _ := config["workflowId"].(string)
	namespace, _ := config["namespace"].(string)
	if namespace == "" {
		// Default to current namespace (passed via config or context)
		namespace, _ = config["_namespace"].(string)
	}

	// 1. Start Child Workflow
	childExecID, err := s.engine.StartWorkflow(ctx, namespace, childWFID, input)
	if err != nil {
		return "", fmt.Errorf("failed to start sub-workflow %s: %w", childWFID, err)
	}

	// 2. Poll for Completion (Simplified Child Workflow Pattern)
	// In a real high-load system, this would be handled by event triggers, 
	// but for our minimalist engine, polling with backoff is robust.
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
			exec, err := s.engine.GetExecutionStatus(ctx, namespace, childExecID)
			if err != nil {
				return "", err
			}

			if exec.Status == models.TaskCompleted {
				return exec.Output, nil
			}
			if exec.Status == models.TaskFailed {
				return "", fmt.Errorf("sub-workflow %s failed: %s", childExecID, exec.ErrorMessage)
			}
			if exec.Status == models.TaskCancelled {
				return "", fmt.Errorf("sub-workflow %s was cancelled", childExecID)
			}
			
			// Continue polling...
		}
	}
}

func (s *SubWorkflowNode) Validate(config map[string]interface{}) error {
	if _, ok := config["workflowId"].(string); !ok {
		return fmt.Errorf("missing 'workflowId' in sub-workflow config")
	}
	return nil
}
