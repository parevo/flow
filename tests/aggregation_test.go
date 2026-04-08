package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/storage/memory"
)

func TestSmartAggregation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("mock", &MockNode{})
	eng := engine.NewEngine(memStore, reg)

	// Define a workflow with parallel paths merging into a single node
	wf := &models.Workflow{
		ID: "agg-wf",
		Nodes: []models.Node{
			{ID: "start", Type: "mock", Name: "Start"},
			{ID: "branch1", Type: "mock", Name: "Branch 1"},
			{ID: "branch2", Type: "mock", Name: "Branch 2"},
			{ID: "join", Type: "mock", Name: "Join Node"},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "start", TargetID: "branch1"},
			{ID: "e2", SourceID: "start", TargetID: "branch2"},
			{ID: "e3", SourceID: "branch1", TargetID: "join"},
			{ID: "e4", SourceID: "branch2", TargetID: "join"},
		},
	}
	memStore.SaveWorkflow(ctx, "test", wf)

	// Start execution
	execID, _ := eng.StartWorkflow(ctx, "test", "agg-wf", `{"initial": "data"}`)

	// Start worker
	w := engine.NewWorker("w1", eng, reg, 50*time.Millisecond)
	go w.Start(ctx)

	// Wait for completion
	deadline := time.Now().Add(5 * time.Second)
	var finalInput string
	for time.Now().Before(deadline) {
		steps, _ := eng.GetExecutionSteps(ctx, "test", execID)
		for _, s := range steps {
			if s.NodeID == "join" && s.Status != models.TaskPending {
				finalInput = s.Input
				break
			}
		}
		if finalInput != "" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if finalInput == "" {
		t.Fatal("Workflow failed to reach join node or join node input was empty")
	}

	// Verify aggregation
	var agg map[string]interface{}
	if err := json.Unmarshal([]byte(finalInput), &agg); err != nil {
		t.Fatalf("Join node input is not valid JSON aggregation: %v", err)
	}

	if _, ok := agg["branch1"]; !ok {
		t.Error("Aggregation missing branch1 output")
	}
	if _, ok := agg["branch2"]; !ok {
		t.Error("Aggregation missing branch2 output")
	}

	t.Logf("Successfully aggregated data: %s", finalInput)
}
