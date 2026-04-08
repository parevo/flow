package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/storage/memory"
)

func TestChaosResilience(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("mock", &MockNode{})
	eng := engine.NewEngine(memStore, reg)

	t.Run("Worker Crash and Zombie Recovery", func(t *testing.T) {
		// Set threshold to 1s for fast testing
		memStore.ZombieThreshold = 1 * time.Second

		wf := &models.Workflow{
			ID: "resilient-wf",
			Nodes: []models.Node{
				{ID: "step1", Type: "mock", Name: "Step 1"},
				{ID: "step2", Type: "mock", Name: "Step 2"},
			},
			Edges: []models.Edge{
				{ID: "e1", SourceID: "step1", TargetID: "step2"},
			},
		}
		memStore.SaveWorkflow(ctx, "chaos", wf)

		// Start execution
		execID, _ := eng.StartWorkflow(ctx, "chaos", "resilient-wf", "input")

		// Create a worker that will "crash" after starting step 1
		_ = engine.NewWorker("w-crashing", eng, reg, 50*time.Millisecond)
		
		// Manually claim and mark as running to simulate a crash
		// instead of full goroutine management for simplicity
		steps, _ := eng.GetExecutionSteps(ctx, "chaos", execID)
		var step1 models.ExecutionStep
		for _, s := range steps {
			if s.NodeID == "step1" {
				step1 = *s
				break
			}
		}

		step1.Status = models.TaskRunning
		step1.WorkerID = "w-dead"
		step1.UpdatedAt = time.Now().Add(-2 * time.Second) // Set back in time so it's a zombie
		memStore.UpdateExecutionStep(ctx, "chaos", &step1)

		// Now start a healthy worker
		w2 := engine.NewWorker("w-survivor", eng, reg, 50*time.Millisecond)
		go w2.Start(ctx)

		// Wait for completion
		deadline := time.Now().Add(10 * time.Second)
		success := false
		for time.Now().Before(deadline) {
			exec, _ := eng.GetExecutionStatus(ctx, "chaos", execID)
			if exec.Status == models.TaskCompleted {
				success = true
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		if !success {
			t.Fatal("Workflow failed to recover from 'dead' worker and complete")
		}
		
		// Verify step 1 was eventually processed by w-survivor
		finalSteps, _ := eng.GetExecutionSteps(ctx, "chaos", execID)
		for _, s := range finalSteps {
			if s.NodeID == "step1" && s.WorkerID != "w-survivor" {
				t.Errorf("Step 1 was NOT recovered by survivor worker, workerID: %s", s.WorkerID)
			}
		}
	})

	t.Run("High Concurrency Stress Test", func(t *testing.T) {
		// 100 parallel workflows
		count := 100
		var wg sync.WaitGroup
		wg.Add(count)

		wf := &models.Workflow{
			ID: "stress-wf",
			Nodes: []models.Node{
				{ID: "s1", Type: "mock", Name: "Fast Step"},
			},
		}
		memStore.SaveWorkflow(ctx, "stress", wf)

		// 5 Workers handling the load
		for i := 0; i < 5; i++ {
			w := engine.NewWorker(fmt.Sprintf("stress-w-%d", i), eng, reg, 10*time.Millisecond)
			go w.Start(ctx)
		}

		execIDs := make([]string, count)
		for i := 0; i < count; i++ {
			id, _ := eng.StartWorkflow(ctx, "stress", "stress-wf", "")
			execIDs[i] = id
		}

		// Check for completion of all
		deadline := time.Now().Add(15 * time.Second)
		completedCount := 0
		
		for time.Now().Before(deadline) {
			completedCount = 0
			for _, id := range execIDs {
				ex, _ := eng.GetExecutionStatus(ctx, "stress", id)
				if ex.Status == models.TaskCompleted {
					completedCount++
				}
			}
			if completedCount == count {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}

		if completedCount < count {
			t.Fatalf("Stress test failed: only %d/%d completed", completedCount, count)
		}
	})
}
