package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/node"
	"github.com/parevo/flow/internal/storage/memory"
	"github.com/parevo/flow/internal/trigger"
)

func TestSequentialOrderProcessing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("mock", &MockNode{})
	reg.Register("fail", &FailureNode{FailUntil: 10}) // Will fail until 10 attempts
	reg.Register("compensation", &MockNode{Name: "CANCEL_ORDER"})
	
	eng := engine.NewEngine(memStore, reg)
	worker := engine.NewWorker("worker-1", eng, reg, 100*time.Millisecond)

	// Define Flow: Order -> Payment (Fails) -> Compensation (Triggered)
	wf := &models.Workflow{
		ID: "seq-order",
		Nodes: []models.Node{
			{ID: "order", Type: "mock", Name: "OrderReceived"},
			{
				ID: "payment", 
				Type: "fail", 
				Name: "PaymentCheck",
				RetryPolicy: &models.RetryPolicy{
					MaxAttempts: 2, // Only 2 attempts, then fail definitively
					InitialInterval: 10 * time.Millisecond,
					BackoffCoefficient: 1.0,
				},
				CompensateNodeID: "rollback",
			},
			{ID: "rollback", Type: "compensation"},
			{ID: "approve", Type: "mock", Name: "OrderApproved"},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "order", TargetID: "payment"},
			{ID: "e2", SourceID: "payment", TargetID: "approve"},
		},
	}
	memStore.SaveWorkflow(ctx, "ns1", wf)

	go worker.Start(ctx)

	execID, _ := eng.StartWorkflow(ctx, "ns1", "seq-order", "data")

	// Wait for failure and compensation
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		exec, _ := memStore.GetExecution(ctx, "ns1", execID)
		if exec != nil && exec.Status == models.TaskFailed {
			steps, _ := memStore.GetExecutionSteps(ctx, "ns1", execID)
			
			foundRollback := false
			paymentAttempts := 0
			for _, s := range steps {
				if s.NodeID == "rollback" && s.Status == models.TaskCompleted {
					foundRollback = true
				}
				if s.NodeID == "payment" {
					paymentAttempts = s.AttemptNumber
				}
			}

			if paymentAttempts != 2 {
				t.Fatalf("Expected 2 retry attempts, got %d", paymentAttempts)
			}
			if !foundRollback {
				t.Fatal("Saga compensation node was not executed")
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("Workflow did not fail as expected")
}

func TestParallelProductionAndInventory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("mock", &MockNode{})
	reg.Register("sleep", &SleepNode{Duration: 500 * time.Millisecond})
	
	eng := engine.NewEngine(memStore, reg)
	worker1 := engine.NewWorker("w1", eng, reg, 50*time.Millisecond)
	worker2 := engine.NewWorker("w2", eng, reg, 50*time.Millisecond)

	wf := &models.Workflow{
		ID: "parallel-flow",
		Nodes: []models.Node{
			{ID: "start", Type: "mock", Name: "StartProduction"},
			{ID: "inventory", Type: "sleep"}, // Parallel branch 1
			{ID: "shipping", Type: "sleep"},  // Parallel branch 2
			{ID: "merge", Type: "mock", Name: "MergeResults"},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "start", TargetID: "inventory"},
			{ID: "e2", SourceID: "start", TargetID: "shipping"},
			{ID: "e3", SourceID: "inventory", TargetID: "merge"},
			{ID: "e4", SourceID: "shipping", TargetID: "merge"},
		},
	}
	memStore.SaveWorkflow(ctx, "ns2", wf)

	go worker1.Start(ctx)
	go worker2.Start(ctx)

	execID, _ := eng.StartWorkflow(ctx, "ns2", "parallel-flow", "input")

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		exec, _ := memStore.GetExecution(ctx, "ns2", execID)
		if exec != nil && exec.Status == models.TaskCompleted {
			steps, _ := memStore.GetExecutionSteps(ctx, "ns2", execID)
			
			var startT, inventoryT, shippingT, mergeT time.Time
			for _, s := range steps {
				if s.NodeID == "start" { startT = *s.FinishedAt }
				if s.NodeID == "inventory" { inventoryT = s.StartedAt }
				if s.NodeID == "shipping" { shippingT = s.StartedAt }
				if s.NodeID == "merge" { mergeT = s.StartedAt }
			}

			// Verify Parallelism: both started after 'start' finished
			if inventoryT.Before(startT) || shippingT.Before(startT) {
				t.Fatal("Parallel nodes started before parent finished")
			}
			// Verify Merge: only started after both finished
			if mergeT.Before(inventoryT.Add(500*time.Millisecond)) || mergeT.Before(shippingT.Add(500*time.Millisecond)) {
				t.Fatal("Merge node started before parallel predecessors finished")
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("Parallel workflow did not complete")
}

func TestHumanInTheLoopSignal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("mock", &MockNode{})
	reg.Register("signal", &node.SignalNode{})
	
	eng := engine.NewEngine(memStore, reg)
	worker := engine.NewWorker("worker-sig", eng, reg, 50*time.Millisecond)

	wf := &models.Workflow{
		ID: "signal-wf",
		Nodes: []models.Node{
			{ID: "check", Type: "mock", Name: "PaymentOK"},
			{ID: "wait-approval", Type: "signal"},
			{ID: "ship", Type: "mock", Name: "ShipOrder"},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "check", TargetID: "wait-approval"},
			{ID: "e2", SourceID: "wait-approval", TargetID: "ship"},
		},
	}
	memStore.SaveWorkflow(ctx, "ns3", wf)

	go worker.Start(ctx)

	execID, _ := eng.StartWorkflow(ctx, "ns3", "signal-wf", "input")

	// 1. Verify WAITING state
	deadline := time.Now().Add(3 * time.Second)
	signaled := false
	for time.Now().Before(deadline) {
		steps, _ := memStore.GetExecutionSteps(ctx, "ns3", execID)
		for _, s := range steps {
			if s.NodeID == "wait-approval" && s.Status == models.TaskWaiting {
				// Successully reached WAITING. Now SIGNAL it.
				fmt.Println("[TEST] Workflow reached WAITING state. Signaling now...")
				err := eng.SignalExecution(ctx, "ns3", execID, "wait-approval", "APPROVED_BY_MANAGER")
				if err != nil {
					t.Fatalf("Failed to signal: %v", err)
				}
				signaled = true
				break
			}
		}
		if signaled { break }
		time.Sleep(200 * time.Millisecond)
	}

	if !signaled { t.Fatal("Workflow never reached WAITING state") }

	// 2. Verify Completion
	for time.Now().Before(deadline) {
		exec, _ := memStore.GetExecution(ctx, "ns3", execID)
		if exec.Status == models.TaskCompleted {
			steps, _ := memStore.GetExecutionSteps(ctx, "ns3", execID)
			for _, s := range steps {
				if s.NodeID == "ship" {
					if s.Input != "APPROVED_BY_MANAGER" {
						t.Fatalf("Ship node did not receive signal output. Got: %s", s.Input)
					}
					return
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("Workflow did not complete after signal")
}

func TestPeriodicReportingCron(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("count", &CountingNode{Counter: new(int), mu: &sync.Mutex{}})
	
	eng := engine.NewEngine(memStore, reg)
	worker := engine.NewWorker("worker-cron", eng, reg, 50*time.Millisecond)
	go worker.Start(ctx)

	wf := &models.Workflow{
		ID: "cron-wf",
		Nodes: []models.Node{
			{ID: "report", Type: "count"},
		},
		Edges: []models.Edge{},
	}
	memStore.SaveWorkflow(ctx, "ns4", wf)

	cronMgr := trigger.NewCronManager(eng, eng.GetLogger())
	// Test cron: every second
	_, err := cronMgr.AddSchedule("ns4", "cron-wf", "* * * * * *", "auto-input")
	if err != nil {
		t.Fatalf("Failed to add cron: %v", err)
	}
	cronMgr.Start()
	defer cronMgr.Stop()

	// Wait for at least 2 executions
	time.Sleep(3 * time.Second)

	executions, _ := memStore.ListExecutions(ctx, "ns4")
	if len(executions) < 2 {
		t.Fatalf("Expected at least 2 cron executions, got %d", len(executions))
	}
}

func TestSubWorkflowOrchestration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("mock", &MockNode{})
	
	eng := engine.NewEngine(memStore, reg)
	// Register SubWorkflowNode specifically with our engine
	reg.Register("subwf", node.NewSubWorkflowNode(eng))
	
	worker1 := engine.NewWorker("worker-sub-1", eng, reg, 50*time.Millisecond)
	worker2 := engine.NewWorker("worker-sub-2", eng, reg, 50*time.Millisecond)
	go worker1.Start(ctx)
	go worker2.Start(ctx)

	// 1. Define Child Workflow
	childWF := &models.Workflow{
		ID: "child-wf",
		Nodes: []models.Node{
			{ID: "step1", Type: "mock", Name: "ChildStep"},
		},
	}
	memStore.SaveWorkflow(ctx, "ns5", childWF)

	// 2. Define Parent Workflow
	parentWF := &models.Workflow{
		ID: "parent-wf",
		Nodes: []models.Node{
			{ID: "start", Type: "mock", Name: "ParentStart"},
			{ID: "run-child", Type: "subwf", Config: map[string]interface{}{
				"workflowId": "child-wf",
				"namespace": "ns5",
			}},
			{ID: "finish", Type: "mock", Name: "ParentFinish"},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "start", TargetID: "run-child"},
			{ID: "e2", SourceID: "run-child", TargetID: "finish"},
		},
	}
	memStore.SaveWorkflow(ctx, "ns5", parentWF)

	execID, _ := eng.StartWorkflow(ctx, "ns5", "parent-wf", "initial-data")

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		exec, _ := memStore.GetExecution(ctx, "ns5", execID)
		if exec != nil && exec.Status == models.TaskCompleted {
			// Success! Check if child was actually called
			childExecs, _ := memStore.ListExecutions(ctx, "ns5")
			foundChild := false
			for _, ce := range childExecs {
				if ce.WorkflowID == "child-wf" { foundChild = true }
			}
			if !foundChild { t.Fatal("Child workflow was never executed") }
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("Parent workflow did not complete")
}
