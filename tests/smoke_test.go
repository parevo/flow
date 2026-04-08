package tests

import (
	"context"
	"testing"
	"time"

	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/node"
	"github.com/parevo/flow/internal/storage/memory"
)

func TestFullWorkflowExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Setup
	storage := memory.NewMemoryStorage()
	registry := engine.NewRegistry()
	registry.Register("log", node.NewLogNode())

	eng := engine.NewEngine(storage, registry)
	worker := engine.NewWorker("worker-1", eng, registry, 100*time.Millisecond)

	namespace := "test-org"

	// 2. Create a simple workflow: node-1 -> node-2
	wf := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []models.Node{
			{ID: "node-1", Type: "log", Config: map[string]interface{}{"message": "Hello from Node 1"}},
			{ID: "node-2", Type: "log", Config: map[string]interface{}{"message": "Hello from Node 2"}},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "node-1", TargetID: "node-2"},
		},
	}
	storage.SaveWorkflow(ctx, namespace, wf)

	// 3. Start Execution
	execID, err := eng.StartWorkflow(ctx, namespace, "wf-1", `{"initial": "input"}`)
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}

	// 4. Run Worker in background
	go worker.Start(ctx)

	// 5. Poll for completion
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		exec, err := storage.GetExecution(ctx, namespace, execID)
		if err == nil && exec.Status == models.TaskCompleted {
			return // Success!
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatal("Workflow timed out or failed")
}
