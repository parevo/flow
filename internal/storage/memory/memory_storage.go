package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/parevo/flow/internal/models"
)

type MemoryStorage struct {
	mu             sync.RWMutex
	workflows      map[string]*models.Workflow
	executions     map[string]*models.Execution
	executionSteps map[string]*models.ExecutionStep
	ZombieThreshold time.Duration
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		workflows:      make(map[string]*models.Workflow),
		executions:     make(map[string]*models.Execution),
		executionSteps: make(map[string]*models.ExecutionStep),
		ZombieThreshold: 5 * time.Minute,
	}
}

func (s *MemoryStorage) SaveWorkflow(ctx context.Context, namespace string, wf *models.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	wf.Namespace = namespace
	key := fmt.Sprintf("%s:%s", namespace, wf.ID)
	s.workflows[key] = wf
	return nil
}

func (s *MemoryStorage) GetWorkflow(ctx context.Context, namespace string, id string) (*models.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", namespace, id)
	wf, ok := s.workflows[key]
	if !ok {
		return nil, fmt.Errorf("workflow not found in namespace %s", namespace)
	}
	return wf, nil
}

func (s *MemoryStorage) ListWorkflows(ctx context.Context, namespace string) ([]*models.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var wfs []*models.Workflow
	for _, wf := range s.workflows {
		if wf.Namespace == namespace {
			wfs = append(wfs, wf)
		}
	}
	return wfs, nil
}

func (s *MemoryStorage) CreateExecution(ctx context.Context, namespace string, exec *models.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	exec.Namespace = namespace
	key := fmt.Sprintf("%s:%s", namespace, exec.ID)
	s.executions[key] = exec
	return nil
}

func (s *MemoryStorage) GetExecution(ctx context.Context, namespace string, id string) (*models.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", namespace, id)
	exec, ok := s.executions[key]
	if !ok {
		return nil, fmt.Errorf("execution not found in namespace %s", namespace)
	}
	// Return a copy to avoid race conditions when execution is updated by workers
	execCopy := *exec
	return &execCopy, nil
}

func (s *MemoryStorage) UpdateExecution(ctx context.Context, namespace string, exec *models.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s:%s", namespace, exec.ID)
	s.executions[key] = exec
	return nil
}

func (s *MemoryStorage) ListExecutions(ctx context.Context, namespace string) ([]*models.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var execs []*models.Execution
	for _, exec := range s.executions {
		if exec.Namespace == namespace {
			execs = append(execs, exec)
		}
	}
	return execs, nil
}

func (s *MemoryStorage) CreateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	step.Namespace = namespace
	step.UpdatedAt = time.Now()
	key := fmt.Sprintf("%s:%s", namespace, step.ID)
	s.executionSteps[key] = step
	return nil
}

func (s *MemoryStorage) UpdateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s:%s", namespace, step.ID)
	step.UpdatedAt = time.Now()
	s.executionSteps[key] = step
	return nil
}

func (s *MemoryStorage) GetExecutionSteps(ctx context.Context, namespace string, executionID string) ([]*models.ExecutionStep, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var steps []*models.ExecutionStep
	for _, step := range s.executionSteps {
		if step.ExecutionID == executionID && step.Namespace == namespace {
			// Return a copy to avoid race conditions when steps are updated by workers
			stepCopy := *step
			steps = append(steps, &stepCopy)
		}
	}
	return steps, nil
}

func (s *MemoryStorage) ClaimReadyStep(ctx context.Context, namespace string, workerID string) (*models.ExecutionStep, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, step := range s.executionSteps {
		isZombie := step.Status == models.TaskRunning && s.ZombieThreshold > 0 && time.Since(step.UpdatedAt) > s.ZombieThreshold

		if step.Status == models.TaskPending || isZombie {
			if namespace == "" || step.Namespace == namespace {
				if step.Status == models.TaskPending && (step.ScheduledAt != nil && step.ScheduledAt.After(now)) {
					continue
				}

				step.Status = models.TaskRunning
				step.WorkerID = workerID
				step.UpdatedAt = now // Refresh heart-beat
				return step, nil
			}
		}
	}
	return nil, nil
}
