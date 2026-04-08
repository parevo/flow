package engine

import (
	"context"
	"testing"
	"time"

	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/storage/memory"
)

func TestFullWorkflowExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Setup
	store := memory.NewMemoryStorage()
	reg := NewRegistry()
	reg.Register("log", &LogNode{})
	eng := NewEngine(store, reg)
	worker := NewWorker("worker-1", eng, reg, 100*time.Millisecond)

	namespace := "test-org"

	// 2. Define a simple 2-node workflow: A -> B
	wf := &models.Workflow{
		ID:      "test-wf",
		Name:    "Test Workflow",
		Version: 1,
		Status:  models.WorkflowActive,
		Nodes: []models.Node{
			{ID: "node-1", Type: "log", Config: map[string]interface{}{"message": "Hello from Node 1"}},
			{ID: "node-2", Type: "log", Config: map[string]interface{}{"message": "Hello from Node 2"}},
		},
		Edges: []models.Edge{
			{ID: "edge-1", SourceID: "node-1", TargetID: "node-2"},
		},
	}
	store.SaveWorkflow(ctx, namespace, wf)

	// 3. Start Workflow
	execID, err := eng.StartWorkflow(ctx, namespace, "test-wf", "Initial Input")
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}

	// 4. Run Worker in background
	go worker.Start(ctx)

	// 5. Poll for completion
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		exec, _ := store.GetExecution(ctx, namespace, execID)
		if exec != nil && exec.Status == models.TaskCompleted {
			// Success!
			steps, _ := store.GetExecutionSteps(ctx, namespace, execID)
			if len(steps) != 2 {
				t.Fatalf("Expected 2 steps to be completed, got %d", len(steps))
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatal("Workflow did not complete in time")
}

func TestNamespaceIsolation(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemoryStorage()
	reg := NewRegistry()
	eng := NewEngine(store, reg)

	wf := &models.Workflow{ID: "isol-wf", Name: "Isolation Test"}
	store.SaveWorkflow(ctx, "org-a", wf)

	// Try to get from org-b
	_, err := eng.storage.GetWorkflow(ctx, "org-b", "isol-wf")
	if err == nil {
		t.Fatal("Org-B should not be able to see Org-A's workflow")
	}
}
