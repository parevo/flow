package flow_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parevo/flow"
)

// ============================================
// STORAGE TESTS
// ============================================

func TestMemoryStorage(t *testing.T) {
	storage := flow.NewMemoryStorage()
	if storage == nil {
		t.Fatal("NewMemoryStorage returned nil")
	}

	ctx := context.Background()

	// Test workflow save and retrieve
	wf := &flow.Workflow{
		ID:      "test-wf",
		Name:    "Test Workflow",
		Version: 1,
		Nodes:   []flow.Node{{ID: "node1", Type: "function"}},
		Edges:   []flow.Edge{},
	}

	err := storage.SaveWorkflow(ctx, "default", wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	retrieved, err := storage.GetWorkflow(ctx, "default", "test-wf")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if retrieved.ID != wf.ID {
		t.Errorf("Expected workflow ID %s, got %s", wf.ID, retrieved.ID)
	}

	// Test list workflows
	workflows, err := storage.ListWorkflows(ctx, "default")
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
	if len(workflows) == 0 {
		t.Error("Expected at least one workflow")
	}
}

func TestMemoryStorageExecutions(t *testing.T) {
	storage := flow.NewMemoryStorage()
	ctx := context.Background()

	exec := &flow.Execution{
		ID:         "exec-123",
		WorkflowID: "wf-1",
		Version:    1,
		Status:     flow.StatusPending,
		Input:      `{"test": true}`,
		StartedAt:  time.Now(),
	}

	// Create
	err := storage.CreateExecution(ctx, "default", exec)
	if err != nil {
		t.Fatalf("CreateExecution failed: %v", err)
	}

	// Get
	retrieved, err := storage.GetExecution(ctx, "default", "exec-123")
	if err != nil {
		t.Fatalf("GetExecution failed: %v", err)
	}
	if retrieved.ID != exec.ID {
		t.Errorf("Expected execution ID %s, got %s", exec.ID, retrieved.ID)
	}

	// Update
	exec.Status = flow.StatusCompleted
	exec.Output = `{"result": "success"}`
	err = storage.UpdateExecution(ctx, "default", exec)
	if err != nil {
		t.Fatalf("UpdateExecution failed: %v", err)
	}

	// List
	executions, err := storage.ListExecutions(ctx, "default")
	if err != nil {
		t.Fatalf("ListExecutions failed: %v", err)
	}
	if len(executions) == 0 {
		t.Error("Expected at least one execution")
	}
}

func TestMemoryStorageExecutionSteps(t *testing.T) {
	storage := flow.NewMemoryStorage()
	ctx := context.Background()

	step := &flow.ExecutionStep{
		ID:          "step-123",
		ExecutionID: "exec-123",
		NodeID:      "node1",
		Status:      flow.StatusPending,
		Input:       `{"test": true}`,
		StartedAt:   time.Now(),
	}

	// Create
	err := storage.CreateExecutionStep(ctx, "default", step)
	if err != nil {
		t.Fatalf("CreateExecutionStep failed: %v", err)
	}

	// Update
	step.Status = flow.StatusCompleted
	step.Output = `{"done": true}`
	err = storage.UpdateExecutionStep(ctx, "default", step)
	if err != nil {
		t.Fatalf("UpdateExecutionStep failed: %v", err)
	}

	// Get steps
	steps, err := storage.GetExecutionSteps(ctx, "default", "exec-123")
	if err != nil {
		t.Fatalf("GetExecutionSteps failed: %v", err)
	}
	if len(steps) == 0 {
		t.Error("Expected at least one step")
	}

	// Claim ready step
	step2 := &flow.ExecutionStep{
		ID:          "step-456",
		ExecutionID: "exec-456",
		NodeID:      "node2",
		Status:      flow.StatusPending,
		StartedAt:   time.Now(),
	}
	storage.CreateExecutionStep(ctx, "default", step2)

	claimed, err := storage.ClaimReadyStep(ctx, "default", "worker-1")
	if err != nil {
		t.Logf("ClaimReadyStep: %v (may be expected)", err)
	}
	if claimed != nil {
		t.Logf("Claimed step: %s", claimed.ID)
	}
}

func TestRedisStorage(t *testing.T) {
	storage := flow.NewRedisStorage("localhost:6379", "", 0)
	if storage == nil {
		t.Fatal("NewRedisStorage returned nil")
	}
}

// ============================================
// REGISTRY TESTS
// ============================================

func TestRegistry(t *testing.T) {
	registry := flow.NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestRegistryFunctions(t *testing.T) {
	registry := flow.NewRegistry()

	registry.RegisterFunction("TestFunc", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"status": "ok"}, nil
	})

	// Create engine to trigger registration
	storage := flow.NewMemoryStorage()
	engine := flow.NewEngine(storage, registry)
	_ = engine

	t.Log("Function registered successfully")
}

func TestRegistryMultipleFunctions(t *testing.T) {
	registry := flow.NewRegistry()

	for i := 0; i < 10; i++ {
		name := "Func" + string(rune(i+48))
		registry.RegisterFunction(name, func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			return input, nil
		})
	}

	t.Log("Registered multiple functions")
}

// ============================================
// ENGINE TESTS
// ============================================

func TestEngineCreation(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}

	logger := engine.GetLogger()
	if logger == nil {
		t.Error("GetLogger returned nil")
	}
}

func TestEngineWithLogger(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	customLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	engine.WithLogger(customLogger)

	if engine.GetLogger() != customLogger {
		t.Error("Custom logger not set")
	}
}

func TestEngineRegisterWorkflow(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "test-register",
		Name:    "Test Register Workflow",
		Version: 1,
		Nodes:   []flow.Node{{ID: "start", Type: "function"}},
		Edges:   []flow.Edge{},
	}

	ctx := context.Background()
	err := engine.RegisterWorkflow(ctx, "default", wf)
	if err != nil {
		t.Fatalf("RegisterWorkflow failed: %v", err)
	}
}

func TestEngineExecuteWorkflow(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()

	executed := false
	registry.RegisterFunction("Task1", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		executed = true
		return map[string]interface{}{"done": true}, nil
	})

	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "exec-test",
		Name:    "Execution Test",
		Version: 1,
		Nodes: []flow.Node{
			{ID: "task1", Type: "function", Config: map[string]interface{}{"function": "Task1"}},
		},
		Edges: []flow.Edge{},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	workerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	go engine.StartWorker(workerCtx, "default", "worker-exec")

	execID, err := engine.Execute(ctx, "default", "exec-test", []byte(`{"input":"test"}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if execID == "" {
		t.Error("Execute returned empty execution ID")
	}

	time.Sleep(500 * time.Millisecond)

	exec, err := engine.GetExecution(ctx, "default", execID)
	if err != nil {
		t.Fatalf("GetExecution failed: %v", err)
	}

	t.Logf("Execution status: %s, Task executed: %v", exec.Status, executed)

	if !executed {
		t.Error("Task was not executed")
	}
}

func TestEngineGetExecutionSteps(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	registry.RegisterFunction("StepTask", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"step": 1}, nil
	})

	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "steps-test",
		Name:    "Steps Test",
		Version: 1,
		Nodes:   []flow.Node{{ID: "step1", Type: "function", Config: map[string]interface{}{"function": "StepTask"}}},
		Edges:   []flow.Edge{},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	workerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	go engine.StartWorker(workerCtx, "default", "worker-steps")

	execID, _ := engine.Execute(ctx, "default", "steps-test", []byte(`{}`))
	time.Sleep(300 * time.Millisecond)

	steps, err := engine.GetExecutionSteps(ctx, "default", execID)
	if err != nil {
		t.Fatalf("GetExecutionSteps failed: %v", err)
	}

	if len(steps) == 0 {
		t.Error("Expected at least one step")
	}
}

func TestEngineCancelExecution(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "cancel-test",
		Name:    "Cancel Test",
		Version: 1,
		Nodes:   []flow.Node{{ID: "task", Type: "function"}},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)
	execID, _ := engine.Execute(ctx, "default", "cancel-test", []byte(`{}`))

	time.Sleep(50 * time.Millisecond)

	err := engine.CancelExecution(ctx, "default", execID)
	if err != nil {
		t.Logf("CancelExecution: %v (may be expected)", err)
	}
}

func TestEngineFailExecution(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "fail-test",
		Name:    "Fail Test",
		Version: 1,
		Nodes:   []flow.Node{{ID: "task", Type: "function"}},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)
	execID, _ := engine.Execute(ctx, "default", "fail-test", []byte(`{}`))

	time.Sleep(50 * time.Millisecond)

	err := engine.FailExecution(ctx, "default", execID, "Test failure")
	if err != nil {
		t.Logf("FailExecution: %v (may be expected)", err)
	}
}

func TestEngineSignalExecution(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "signal-test",
		Name:    "Signal Test",
		Version: 1,
		Nodes:   []flow.Node{{ID: "wait", Type: "signal"}},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)
	execID, _ := engine.Execute(ctx, "default", "signal-test", []byte(`{}`))

	time.Sleep(100 * time.Millisecond)

	err := engine.SignalExecution(ctx, "default", execID, "wait", `{"approved":true}`)
	if err != nil {
		t.Logf("SignalExecution: %v", err)
	}
}

func TestEngineReapTimeouts(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	ctx := context.Background()
	err := engine.ReapTimeouts(ctx, "default")
	if err != nil {
		t.Logf("ReapTimeouts: %v", err)
	}
}

// ============================================
// WORKFLOW BUILDER TESTS
// ============================================

func TestWorkflowBuilder(t *testing.T) {
	builder := flow.NewWorkflow("test-wf", "Test Workflow")
	builder.AddNode("node1", flow.NodeTypeFunction).
		WithConfig("function", "Test1")
	builder.AddNode("node2", flow.NodeTypeFunction).
		WithConfig("function", "Test2")
	builder.Connect("node1", "node2")

	wf := builder.Build()

	if wf.ID != "test-wf" {
		t.Errorf("Expected ID test-wf, got %s", wf.ID)
	}
	if len(wf.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(wf.Nodes))
	}
	if len(wf.Edges) == 0 {
		t.Error("Expected edges")
	}
}

func TestWorkflowBuilderWithConfig(t *testing.T) {
	builder := flow.NewWorkflow("config-test", "Config Test")
	builder.AddNode("http", flow.NodeTypeHTTP).
		WithConfig("url", "https://api.example.com").
		WithConfig("method", "POST").
		WithRetry(3)

	wf := builder.Build()

	if len(wf.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(wf.Nodes))
	}

	node := wf.Nodes[0]
	if node.Config["url"] != "https://api.example.com" {
		t.Error("URL config not set")
	}
	if node.RetryCount != 3 {
		t.Errorf("Expected retry 3, got %d", node.RetryCount)
	}
}

func TestWorkflowBuilderConditional(t *testing.T) {
	builder := flow.NewWorkflow("cond-test", "Conditional")
	builder.AddNode("check", flow.NodeTypeCondition).
		WithConfig("expression", "input.amount > 100")
	builder.AddNode("high", flow.NodeTypeFunction)
	builder.AddNode("low", flow.NodeTypeFunction)
	builder.ConnectIf("check", "high", "true")
	builder.ConnectIf("check", "low", "false")

	wf := builder.Build()

	if len(wf.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(wf.Nodes))
	}
	if len(wf.Edges) < 2 {
		t.Error("Expected conditional edges")
	}
}

func TestWorkflowBuilderSaga(t *testing.T) {
	builder := flow.NewWorkflow("saga-test", "Saga Test")
	builder.AddNode("charge", flow.NodeTypeFunction).
		WithSaga("refund")
	builder.AddNode("refund", flow.NodeTypeFunction)

	wf := builder.Build()

	if wf.Nodes[0].CompensateNodeID != "refund" {
		t.Error("Saga compensation not set")
	}
}

func TestWorkflowBuilderVisualize(t *testing.T) {
	builder := flow.NewWorkflow("viz-test", "Visualization Test")
	builder.AddNode("start", flow.NodeTypeFunction)
	builder.AddNode("end", flow.NodeTypeFunction)
	builder.Connect("start", "end")

	diagram := builder.Visualize()
	if diagram == "" {
		t.Error("Visualize returned empty string")
	}
	t.Logf("Diagram length: %d chars", len(diagram))
}

func TestToMermaid(t *testing.T) {
	wf := &flow.Workflow{
		ID:   "mermaid-test",
		Name: "Mermaid Test",
		Nodes: []flow.Node{
			{ID: "start", Type: "function", Name: "Start"},
			{ID: "end", Type: "function", Name: "End"},
		},
		Edges: []flow.Edge{
			{SourceID: "start", TargetID: "end"},
		},
	}

	diagram := flow.ToMermaid(wf)
	if diagram == "" {
		t.Error("ToMermaid returned empty")
	}
	if len(diagram) < 20 {
		t.Error("Diagram too short")
	}
}

// ============================================
// WORKER TESTS
// ============================================

func TestWorker(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()

	var executed int32
	registry.RegisterFunction("WorkerTask", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		atomic.StoreInt32(&executed, 1)
		return map[string]interface{}{"done": true}, nil
	})

	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "worker-test",
		Name:    "Worker Test",
		Version: 1,
		Nodes:   []flow.Node{{ID: "task", Type: "function", Config: map[string]interface{}{"function": "WorkerTask"}}},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	workerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	worker := flow.NewWorker("test-worker", engine, registry, 100*time.Millisecond)
	worker.SetNamespace("default")
	go worker.Start(workerCtx)

	execID, _ := engine.Execute(ctx, "default", "worker-test", []byte(`{}`))
	time.Sleep(500 * time.Millisecond)

	exec, _ := engine.GetExecution(ctx, "default", execID)
	isExecuted := atomic.LoadInt32(&executed)
	t.Logf("Status: %s, Executed: %v", exec.Status, isExecuted == 1)

	if isExecuted == 0 {
		t.Error("Task not executed by worker")
	}
}

// ============================================
// EVENT BUS TESTS
// ============================================

func TestEventBus(t *testing.T) {
	eventBus := flow.NewEventBus()
	if eventBus == nil {
		t.Fatal("NewEventBus returned nil")
	}

	var received int32
	handler := &testEventHandler{
		fn: func(ctx context.Context, event flow.Event) error {
			atomic.StoreInt32(&received, 1)
			return nil
		},
	}

	eventBus.RegisterHandler(flow.EventWorkflowStarted, handler)
	eventBus.Emit(context.Background(), flow.Event{Type: flow.EventWorkflowStarted})

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&received) == 0 {
		t.Error("Event not received")
	}
}

func TestEventBusGlobalHandler(t *testing.T) {
	eventBus := flow.NewEventBus()

	var count int32
	handler := &testEventHandler{
		fn: func(ctx context.Context, event flow.Event) error {
			atomic.AddInt32(&count, 1)
			return nil
		},
	}

	eventBus.RegisterGlobalHandler(handler)

	ctx := context.Background()
	eventBus.Emit(ctx, flow.Event{Type: flow.EventWorkflowStarted})
	eventBus.Emit(ctx, flow.Event{Type: flow.EventWorkflowCompleted})
	eventBus.Emit(ctx, flow.Event{Type: flow.EventStepStarted})

	time.Sleep(200 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	if finalCount != 3 {
		t.Errorf("Expected 3 events, got %d", finalCount)
	}
}

func TestEventBusEmitSync(t *testing.T) {
	eventBus := flow.NewEventBus()

	handler := &testEventHandler{
		fn: func(ctx context.Context, event flow.Event) error {
			return nil
		},
	}

	eventBus.RegisterHandler(flow.EventWorkflowStarted, handler)

	errs := eventBus.EmitSync(context.Background(), flow.Event{Type: flow.EventWorkflowStarted})
	if len(errs) > 0 {
		t.Errorf("EmitSync returned errors: %v", errs)
	}
}

type testEventHandler struct {
	fn func(context.Context, flow.Event) error
}

func (h *testEventHandler) HandleEvent(ctx context.Context, event flow.Event) error {
	if h.fn != nil {
		return h.fn(ctx, event)
	}
	return nil
}

// ============================================
// CRYPTO TESTS
// ============================================

func TestCrypto(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	crypto, err := flow.NewCrypto(key)
	if err != nil {
		t.Fatalf("NewCrypto failed: %v", err)
	}

	plaintext := "sensitive data"
	encrypted, err := crypto.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == plaintext {
		t.Error("Encrypted should differ from plaintext")
	}

	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected %s, got %s", plaintext, decrypted)
	}
}

func TestCryptoInvalidKey(t *testing.T) {
	_, err := flow.NewCrypto("short")
	if err == nil {
		t.Error("Expected error for short key")
	}
}

func TestCryptoEmptyString(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	crypto, _ := flow.NewCrypto(key)

	encrypted, err := crypto.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}

	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != "" {
		t.Errorf("Expected empty string, got %s", decrypted)
	}
}

// ============================================
// GRAPH/DAG TESTS
// ============================================

func TestGraph(t *testing.T) {
	wf := &flow.Workflow{
		ID:   "graph-test",
		Name: "Graph Test",
		Nodes: []flow.Node{
			{ID: "node1", Type: "function"},
			{ID: "node2", Type: "function"},
			{ID: "node3", Type: "function"},
		},
		Edges: []flow.Edge{
			{SourceID: "node1", TargetID: "node2"},
			{SourceID: "node2", TargetID: "node3"},
		},
	}

	graph, err := flow.NewGraph(wf)
	if err != nil {
		t.Fatalf("NewGraph failed: %v", err)
	}

	if graph == nil {
		t.Fatal("Graph is nil")
	}
}

// ============================================
// TRIGGER TESTS
// ============================================

func TestCronManager(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cronMgr := flow.NewCronManager(engine, logger)

	if cronMgr == nil {
		t.Fatal("NewCronManager returned nil")
	}

	cronMgr.Start()
	time.Sleep(100 * time.Millisecond)
	cronMgr.Stop()
}

func TestWebhookManager(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	webhookMgr := flow.NewWebhookManager(engine)
	if webhookMgr == nil {
		t.Fatal("NewWebhookManager returned nil")
	}
}

func TestAPIManager(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	webhookMgr := flow.NewWebhookManager(engine)
	apiMgr := flow.NewAPIManager(webhookMgr)

	if apiMgr == nil {
		t.Fatal("NewAPIManager returned nil")
	}
}

// ============================================
// CONSTANTS TESTS
// ============================================

func TestNodeTypeConstants(t *testing.T) {
	nodeTypes := []string{
		flow.NodeTypeFunction,
		flow.NodeTypeHTTP,
		flow.NodeTypeCondition,
		flow.NodeTypeLog,
		flow.NodeTypeTransform,
		flow.NodeTypeSignal,
		flow.NodeTypeSubWorkflow,
		flow.NodeTypeAI,
		flow.NodeTypeNotify,
		flow.NodeTypeSwitch,
		flow.NodeTypeWait,
		flow.NodeTypeSetVariable,
	}

	for _, nodeType := range nodeTypes {
		if nodeType == "" {
			t.Error("Found empty node type")
		}
	}

	t.Logf("Verified %d node types", len(nodeTypes))
}

func TestStatusConstants(t *testing.T) {
	statuses := []flow.TaskStatus{
		flow.StatusPending,
		flow.StatusRunning,
		flow.StatusCompleted,
		flow.StatusFailed,
		flow.StatusSkipped,
		flow.StatusWaiting,
		flow.StatusCancelled,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Found empty status")
		}
	}

	t.Logf("Verified %d statuses", len(statuses))
}

func TestWorkflowStatusConstants(t *testing.T) {
	if flow.WorkflowActive == "" {
		t.Error("WorkflowActive is empty")
	}
	if flow.WorkflowInactive == "" {
		t.Error("WorkflowInactive is empty")
	}
}

func TestEventTypeConstants(t *testing.T) {
	eventTypes := []flow.EventType{
		flow.EventWorkflowStarted,
		flow.EventWorkflowCompleted,
		flow.EventWorkflowFailed,
		flow.EventWorkflowCancelled,
		flow.EventStepStarted,
		flow.EventStepCompleted,
		flow.EventStepFailed,
		flow.EventStepRetrying,
		flow.EventStepWaitingSignal,
		flow.EventSignalReceived,
		flow.EventCompensationTriggered,
	}

	for _, et := range eventTypes {
		if et == "" {
			t.Error("Found empty event type")
		}
	}

	t.Logf("Verified %d event types", len(eventTypes))
}

// ============================================
// METRICS TESTS
// ============================================

func TestMetrics(t *testing.T) {
	if flow.WorkflowsStarted == nil {
		t.Error("WorkflowsStarted metric is nil")
	}
	if flow.WorkflowsCompleted == nil {
		t.Error("WorkflowsCompleted metric is nil")
	}
	if flow.WorkflowsFailed == nil {
		t.Error("WorkflowsFailed metric is nil")
	}
	if flow.StepsProcessed == nil {
		t.Error("StepsProcessed metric is nil")
	}
	if flow.StepDuration == nil {
		t.Error("StepDuration metric is nil")
	}
	if flow.ActiveWorkers == nil {
		t.Error("ActiveWorkers metric is nil")
	}
}

// ============================================
// ERROR HANDLING TESTS
// ============================================

func TestExecuteNonExistentWorkflow(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	_, err := engine.Execute(context.Background(), "default", "nonexistent", []byte(`{}`))
	if err == nil {
		t.Error("Expected error for nonexistent workflow")
	}
}

func TestGetNonExistentExecution(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	_, err := engine.GetExecution(context.Background(), "default", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent execution")
	}
}

func TestWorkflowWithInvalidNode(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "invalid-test",
		Name:    "Invalid Test",
		Version: 1,
		Nodes:   []flow.Node{{ID: "task", Type: "nonexistent"}},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	_, err := engine.Execute(ctx, "default", "invalid-test", []byte(`{}`))
	if err == nil {
		t.Error("Expected error for invalid node type")
	}
}

// ============================================
// INTEGRATION TESTS
// ============================================

func TestCompleteWorkflowExecution(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()

	var step1Done int32
	var step2Done int32

	registry.RegisterFunction("Step1", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		atomic.StoreInt32(&step1Done, 1)
		return map[string]interface{}{"step1": "done"}, nil
	})

	registry.RegisterFunction("Step2", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		atomic.StoreInt32(&step2Done, 1)
		return map[string]interface{}{"step2": "done"}, nil
	})

	engine := flow.NewEngine(storage, registry)

	builder := flow.NewWorkflow("integration", "Integration Test")
	builder.AddNode("step1", flow.NodeTypeFunction).WithConfig("function", "Step1")
	builder.AddNode("step2", flow.NodeTypeFunction).WithConfig("function", "Step2")
	builder.Connect("step1", "step2")
	wf := builder.Build()

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	workerCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	go engine.StartWorker(workerCtx, "default", "integration-worker")

	execID, err := engine.Execute(ctx, "default", "integration", []byte(`{"test":true}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Wait for completion
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		exec, _ := engine.GetExecution(ctx, "default", execID)
		if exec.Status == flow.StatusCompleted || exec.Status == flow.StatusFailed {
			break
		}
	}

	exec, _ := engine.GetExecution(ctx, "default", execID)
	t.Logf("Final status: %s", exec.Status)

	if atomic.LoadInt32(&step1Done) == 0 {
		t.Error("Step1 not executed")
	}
	if atomic.LoadInt32(&step2Done) == 0 {
		t.Error("Step2 not executed")
	}
}

func TestWorkflowWithRetry(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()

	attempts := 0
	registry.RegisterFunction("FlakyTask", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		attempts++
		if attempts < 3 {
			return nil, fmt.Errorf("temporary failure")
		}
		return map[string]interface{}{"success": true}, nil
	})

	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "retry-test",
		Name:    "Retry Test",
		Version: 1,
		Nodes: []flow.Node{
			{
				ID:   "flaky",
				Type: "function",
				Config: map[string]interface{}{
					"function": "FlakyTask",
				},
				RetryPolicy: &flow.RetryPolicy{
					MaxAttempts:        5,
					InitialInterval:    100 * time.Millisecond,
					BackoffCoefficient: 2.0,
					MaximumInterval:    time.Second,
				},
			},
		},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	workerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	go engine.StartWorker(workerCtx, "default", "retry-worker")

	execID, _ := engine.Execute(ctx, "default", "retry-test", []byte(`{}`))
	time.Sleep(2 * time.Second)

	exec, _ := engine.GetExecution(ctx, "default", execID)
	t.Logf("Status: %s, Attempts: %d", exec.Status, attempts)

	if attempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attempts)
	}
}

func TestConcurrentWorkflows(t *testing.T) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()

	var execCount int32
	registry.RegisterFunction("ConcurrentTask", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		atomic.AddInt32(&execCount, 1)
		time.Sleep(50 * time.Millisecond)
		return map[string]interface{}{"done": true}, nil
	})

	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "concurrent",
		Name:    "Concurrent Test",
		Version: 1,
		Nodes:   []flow.Node{{ID: "task", Type: "function", Config: map[string]interface{}{"function": "ConcurrentTask"}}},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	workerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for i := 0; i < 3; i++ {
		workerID := "worker-" + string(rune(i+48))
		go engine.StartWorker(workerCtx, "default", workerID)
	}

	// Execute 10 workflows concurrently
	for i := 0; i < 10; i++ {
		input, _ := json.Marshal(map[string]interface{}{"id": i})
		_, err := engine.Execute(ctx, "default", "concurrent", input)
		if err != nil {
			t.Errorf("Execute %d failed: %v", i, err)
		}
	}

	time.Sleep(2 * time.Second)
	t.Logf("Total executions: %d", atomic.LoadInt32(&execCount))
}

// ============================================
// BENCHMARKS
// ============================================

func BenchmarkWorkflowExecution(b *testing.B) {
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	registry.RegisterFunction("BenchTask", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return input, nil
	})

	engine := flow.NewEngine(storage, registry)

	wf := &flow.Workflow{
		ID:      "bench",
		Name:    "Benchmark",
		Version: 1,
		Nodes:   []flow.Node{{ID: "task", Type: "function", Config: map[string]interface{}{"function": "BenchTask"}}},
	}

	ctx := context.Background()
	engine.RegisterWorkflow(ctx, "default", wf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(ctx, "default", "bench", []byte(`{}`))
	}
}

func BenchmarkRegistryFunctionCall(b *testing.B) {
	registry := flow.NewRegistry()
	registry.RegisterFunction("Test", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return input, nil
	})

	input := map[string]interface{}{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(input)
	}
}

func BenchmarkBuilderAPI(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := flow.NewWorkflow("bench", "Benchmark")
		builder.AddNode("n1", flow.NodeTypeFunction).WithConfig("function", "T1")
		builder.AddNode("n2", flow.NodeTypeFunction).WithConfig("function", "T2")
		builder.Connect("n1", "n2")
		_ = builder.Build()
	}
}

func BenchmarkCrypto(b *testing.B) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	crypto, _ := flow.NewCrypto(key)
	plaintext := "secret data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, _ := crypto.Encrypt(plaintext)
		_, _ = crypto.Decrypt(encrypted)
	}
}
