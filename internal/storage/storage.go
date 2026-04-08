package storage

import (
	"context"
	"github.com/parevo/flow/internal/models"
)

type Storage interface {
	// Workflow operations
	SaveWorkflow(ctx context.Context, namespace string, wf *models.Workflow) error
	GetWorkflow(ctx context.Context, namespace string, id string) (*models.Workflow, error)
	ListWorkflows(ctx context.Context, namespace string) ([]*models.Workflow, error)

	// Execution operations
	CreateExecution(ctx context.Context, namespace string, exec *models.Execution) error
	GetExecution(ctx context.Context, namespace string, id string) (*models.Execution, error)
	UpdateExecution(ctx context.Context, namespace string, exec *models.Execution) error
	ListExecutions(ctx context.Context, namespace string) ([]*models.Execution, error)

	// ExecutionStep operations
	CreateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error
	UpdateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error
	GetExecutionSteps(ctx context.Context, namespace string, executionID string) ([]*models.ExecutionStep, error)

	// Worker operations
	// ClaimReadyStep finds a step that is PENDING and has all predecessors COMPLETED.
	// Optionally filtered by namespace if needed.
	ClaimReadyStep(ctx context.Context, namespace string, workerID string) (*models.ExecutionStep, error)
}
