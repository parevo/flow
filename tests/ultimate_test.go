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

func TestEnterpriseEndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping complex end-to-end workflow test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("mock", &MockNode{})
	reg.Register("data", &DataNode{})
	reg.Register("condition", &node.ConditionNode{})
	reg.Register("signal", &node.SignalNode{})
	eng := engine.NewEngine(memStore, reg)
	reg.Register("subwf", node.NewSubWorkflowNode(eng))

	// Setup 3 Workers for load balancing
	workers := []*engine.Worker{
		engine.NewWorker("w1", eng, reg, 50*time.Millisecond),
		engine.NewWorker("w2", eng, reg, 50*time.Millisecond),
		engine.NewWorker("w3", eng, reg, 50*time.Millisecond),
	}
	for _, w := range workers {
		go w.Start(ctx)
	}

	// 1. Define Child Workflow (VIP Logistics)
	childWF := &models.Workflow{
		ID:   "vip-logistics",
		Name: "VIP Logistics",
		Nodes: []models.Node{
			{ID: "priority-pkg", Type: "mock", Name: "Priority Packaging"},
			{ID: "prime-express", Type: "mock", Name: "Prime Express Shipping"},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "priority-pkg", TargetID: "prime-express"},
		},
	}
	memStore.SaveWorkflow(ctx, "ns", childWF)

	// 2. Define Parent Workflow
	parentWF := &models.Workflow{
		ID:   "order-master",
		Name: "Order Master",
		Nodes: []models.Node{
			{ID: "recv", Type: "data", Name: "Order Received"},
			{
				ID:   "payment", 
				Type: "data", 
				Name: "Payment Processor",
				RetryPolicy: &models.RetryPolicy{
					MaxAttempts:       2,
					InitialInterval:   100 * time.Millisecond,
					BackoffCoefficient: 2.0,
				},
				CompensateNodeID: "rollback",
			},
			{ID: "rollback", Type: "mock", Name: "Payment Rollback"},
			{
				ID:   "check-vip", 
				Type: "condition", 
				Name: "VIP Checker",
				Config: map[string]interface{}{
					"variable": "is_vip",
					"operator": "==",
					"value":    true,
				},
			},
			{
				ID:   "vip-logistics-node", 
				Type: "subwf", 
				Name: "SubWorkflow VIP",
				Config: map[string]interface{}{
					"workflowId": "vip-logistics",
					"namespace":  "ns",
				},
			},
			{ID: "inventory", Type: "mock", Name: "Standard Inventory"},
			{ID: "shipping", Type: "mock", Name: "Standard Shipping"},
			{ID: "merge", Type: "mock", Name: "Merge Logistics"},
			{
				ID:   "wait-approval", 
				Type: "signal", 
				Name: "Manager Approval",
				Config: map[string]interface{}{
					"timeout": "5s",
				},
			},
			{ID: "ship-final", Type: "mock", Name: "Final Delivery"},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "recv", TargetID: "payment"},
			{ID: "e2", SourceID: "payment", TargetID: "check-vip"},
			// Branching based on condition result
			{ID: "e3", SourceID: "check-vip", TargetID: "vip-logistics-node", Condition: "true"},
			{ID: "e4", SourceID: "check-vip", TargetID: "inventory", Condition: "false"},
			{ID: "e5", SourceID: "check-vip", TargetID: "shipping", Condition: "false"},
			// Merge for normal
			{ID: "e6", SourceID: "inventory", TargetID: "merge"},
			{ID: "e7", SourceID: "shipping", TargetID: "merge"},
			// Join both paths to approval
			{ID: "e8", SourceID: "vip-logistics-node", TargetID: "wait-approval"},
			{ID: "e9", SourceID: "merge", TargetID: "wait-approval"},
			// Final step
			{ID: "e10", SourceID: "wait-approval", TargetID: "ship-final"},
		},
	}
	memStore.SaveWorkflow(ctx, "ns", parentWF)

	// Run timeout reaper every second
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				eng.ReapTimeouts(ctx, "ns")
			}
		}
	}()

	// TEST CASE A: VIP ORDER
	t.Run("VIP Order Execution", func(t *testing.T) {
		input := `{"is_vip": true}`
		execID, err := eng.StartWorkflow(ctx, "ns", "order-master", input)
		if err != nil {
			t.Fatal(err)
		}

		// Wait for signal state
		deadline := time.Now().Add(25 * time.Second)
		var approvalStepID string
		for time.Now().Before(deadline) {
			steps, _ := eng.GetExecutionSteps(ctx, "ns", execID)
			for _, s := range steps {
				if s.NodeID == "wait-approval" && s.Status == models.TaskWaiting {
					approvalStepID = s.ID
					break
				}
			}
			if approvalStepID != "" {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		if approvalStepID == "" {
			t.Fatal("Workflow failed to reach WAITING state")
		}

		// Signal it!
		eng.SignalExecution(ctx, "ns", execID, "wait-approval", "APPROVED")

		// Wait for completion
		deadline = time.Now().Add(25 * time.Second)
		success := false
		for time.Now().Before(deadline) {
			exec, _ := eng.GetExecutionStatus(ctx, "ns", execID)
			if exec.Status == models.TaskCompleted {
				success = true
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		if !success {
			t.Fatal("VIP Workflow failed to complete")
		}
	})

	// TEST CASE B: NORMAL ORDER + SIGNAL TIMEOUT
	t.Run("Normal Order with Timeout", func(t *testing.T) {
		input := `{"is_vip": false}`
		execID, err := eng.StartWorkflow(ctx, "ns", "order-master", input)
		if err != nil {
			t.Fatal(err)
		}

		// Wait for WAITING state, then DO NOT SIGNAL and wait for timeout
		deadline := time.Now().Add(20 * time.Second)
		timedOut := false
		for time.Now().Before(deadline) {
			exec, _ := eng.GetExecutionStatus(ctx, "ns", execID)
			if exec.Status == models.TaskFailed && exec.ErrorMessage == "SIGNAL_TIMEOUT" {
				timedOut = true
				break
			}
			time.Sleep(500 * time.Millisecond)
		}

		if !timedOut {
			t.Fatal("Workflow failed to timeout as expected")
		}
	})
}
