// Package flow provides a high-performance workflow orchestration engine for Go.
//
// Quick Start with MySQL:
//
//	import (
//	    "github.com/parevo/flow"
//	    _ "github.com/go-sql-driver/mysql"
//	    "github.com/jmoiron/sqlx"
//	)
//
//	func main() {
//	    // Connect to MySQL
//	    db, _ := sqlx.Connect("mysql", "user:pass@tcp(localhost:3306)/db?parseTime=true")
//
//	    // Create storage (auto-creates tables!)
//	    storage, _ := flow.NewMySQLStorage(db)
//
//	    // Create engine
//	    registry := flow.NewRegistry()
//	    engine := flow.NewEngine(storage, registry)
//
//	    // Register your functions
//	    registry.RegisterFunction("MyTask", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
//	        return map[string]interface{}{"status": "done"}, nil
//	    })
//
//	    // Start worker
//	    go engine.StartWorker(ctx, "default", "worker-1")
//
//	    // Execute workflow
//	    execID, _ := engine.Execute(ctx, "default", "my-workflow", `{"key":"value"}`)
//	}
package flow

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/parevo/flow/internal/builder"
	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/node"
	"github.com/parevo/flow/internal/storage"
	"github.com/parevo/flow/internal/storage/memory"
	"github.com/parevo/flow/internal/storage/redis"
	"github.com/parevo/flow/internal/storage/sql"
	"github.com/parevo/flow/internal/telemetry"
	"github.com/parevo/flow/internal/trigger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
)

// ========================================
// CORE TYPES
// ========================================

// Engine is the workflow execution engine
type Engine struct {
	*engine.Engine
}

// Registry manages workflow function handlers
type Registry struct {
	*engine.Registry
}

// Storage interface for persistence
type Storage = storage.Storage

// Worker executes workflow tasks
type Worker struct {
	*engine.Worker
}

// Graph represents the DAG structure of a workflow
type Graph = engine.Graph

// ========================================
// WORKFLOW MODELS
// ========================================

type (
	Workflow       = models.Workflow
	Node           = models.Node
	Edge           = models.Edge
	Execution      = models.Execution
	ExecutionStep  = models.ExecutionStep
	RetryPolicy    = models.RetryPolicy
	WorkflowStatus = models.WorkflowStatus
	TaskStatus     = models.TaskStatus
)

// Node types (use these strings in Node.Type field)
const (
	NodeTypeFunction    = "function"
	NodeTypeHTTP        = "http"
	NodeTypeCondition   = "condition"
	NodeTypeLog         = "log"
	NodeTypeTransform   = "transform"
	NodeTypeSignal      = "signal"
	NodeTypeSubWorkflow = "subworkflow"
	NodeTypeAI          = "ai"
	NodeTypeNotify      = "notify"
	NodeTypeSwitch      = "switch"
	NodeTypeWait        = "wait"
	NodeTypeSetVariable = "setvariable"
)

// Workflow statuses
const (
	WorkflowActive   = models.WorkflowActive
	WorkflowInactive = models.WorkflowInactive
)

// Task/Execution statuses
const (
	StatusPending   = models.TaskPending
	StatusRunning   = models.TaskRunning
	StatusCompleted = models.TaskCompleted
	StatusFailed    = models.TaskFailed
	StatusSkipped   = models.TaskSkipped
	StatusWaiting   = models.TaskWaiting
	StatusCancelled = models.TaskCancelled
)

// ========================================
// FUNCTION HANDLER TYPE
// ========================================

// HandlerFunc is the signature for workflow functions
//
// Example:
//
//	func ProcessOrder(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
//	    orderID := input["order_id"].(string)
//	    // ... process order ...
//	    return map[string]interface{}{"status": "processed"}, nil
//	}
type HandlerFunc func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

// NodeExecutor defines the interface for custom node types
type NodeExecutor = engine.NodeExecutor

// ========================================
// ENGINE METHODS
// ========================================

// NewEngine creates a new workflow engine
func NewEngine(storage Storage, registry *Registry) *Engine {
	return &Engine{engine.NewEngine(storage, registry.Registry)}
}

// WithLogger sets a custom logger for the engine
func (e *Engine) WithLogger(l *slog.Logger) *Engine {
	e.Engine.WithLogger(l)
	return e
}

// GetLogger returns the engine's logger
func (e *Engine) GetLogger() *slog.Logger {
	return e.Engine.GetLogger()
}

// RegisterWorkflow saves a workflow definition
func (e *Engine) RegisterWorkflow(ctx context.Context, namespace string, wf *Workflow) error {
	return e.Engine.RegisterWorkflow(ctx, namespace, wf)
}

// Execute starts a new workflow execution (alias for StartWorkflow)
func (e *Engine) Execute(ctx context.Context, namespace, workflowID string, input []byte) (string, error) {
	return e.Engine.StartWorkflow(ctx, namespace, workflowID, string(input))
}

// StartWorkflow starts a new workflow execution
func (e *Engine) StartWorkflow(ctx context.Context, namespace, workflowID string, input string) (string, error) {
	return e.Engine.StartWorkflow(ctx, namespace, workflowID, input)
}

// GetExecution retrieves an execution by ID
func (e *Engine) GetExecution(ctx context.Context, namespace, execID string) (*Execution, error) {
	return e.Engine.GetExecutionStatus(ctx, namespace, execID)
}

// GetExecutionStatus retrieves an execution's status
func (e *Engine) GetExecutionStatus(ctx context.Context, namespace, execID string) (*Execution, error) {
	return e.Engine.GetExecutionStatus(ctx, namespace, execID)
}

// GetExecutionSteps retrieves all steps for an execution
func (e *Engine) GetExecutionSteps(ctx context.Context, namespace, execID string) ([]*ExecutionStep, error) {
	return e.Engine.GetExecutionSteps(ctx, namespace, execID)
}

// CancelExecution cancels a running execution
func (e *Engine) CancelExecution(ctx context.Context, namespace, execID string) error {
	return e.Engine.CancelExecution(ctx, namespace, execID)
}

// FailExecution marks an execution as failed
func (e *Engine) FailExecution(ctx context.Context, namespace, execID, message string) error {
	return e.Engine.FailExecution(ctx, namespace, execID, message)
}

// SignalExecution sends a signal to resume a waiting workflow
func (e *Engine) SignalExecution(ctx context.Context, namespace, execID, nodeID, output string) error {
	return e.Engine.SignalExecution(ctx, namespace, execID, nodeID, output)
}

// ReapTimeouts checks for and handles timed-out tasks
func (e *Engine) ReapTimeouts(ctx context.Context, namespace string) error {
	return e.Engine.ReapTimeouts(ctx, namespace)
}

// StartWorker starts a worker that processes tasks
func (e *Engine) StartWorker(ctx context.Context, namespace, workerID string) {
	e.Engine.StartWorker(ctx, namespace, workerID)
}

// ========================================
// REGISTRY METHODS
// ========================================

// NewRegistry creates a new function registry
func NewRegistry() *Registry {
	return &Registry{engine.NewRegistry()}
}

// RegisterFunction registers a function handler
//
// Example:
//
//	registry.RegisterFunction("SendEmail", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
//	    email := input["email"].(string)
//	    // Send email...
//	    return map[string]interface{}{"sent": true}, nil
//	})
func (r *Registry) RegisterFunction(name string, handler HandlerFunc) {
	r.Registry.RegisterFunction(name, handler)
}

// Register registers a custom node executor
func (r *Registry) Register(nodeType string, executor NodeExecutor) {
	r.Registry.Register(nodeType, executor)
}

// Get retrieves a registered executor by type
func (r *Registry) Get(nodeType string) (NodeExecutor, error) {
	return r.Registry.Get(nodeType)
}

// ========================================
// WORKER
// ========================================

// NewWorker creates a new worker instance
func NewWorker(id string, eng *Engine, registry *Registry, interval time.Duration) *Worker {
	return &Worker{engine.NewWorker(id, eng.Engine, registry.Registry, interval)}
}

// SetNamespace sets the worker's namespace
func (w *Worker) SetNamespace(namespace string) {
	w.Worker.SetNamespace(namespace)
}

// Start begins the worker's task processing loop
func (w *Worker) Start(ctx context.Context) {
	w.Worker.Start(ctx)
}

// ========================================
// STORAGE BACKENDS
// ========================================

// NewMemoryStorage creates an in-memory storage backend (for development)
func NewMemoryStorage() Storage {
	return memory.NewMemoryStorage()
}

// NewMySQLStorage creates a MySQL storage backend with auto-migration
//
// Example:
//
//	db, _ := sqlx.Connect("mysql", "user:pass@tcp(localhost:3306)/db?parseTime=true")
//	storage, _ := flow.NewMySQLStorage(db)
//
// Tables are automatically created if they don't exist.
func NewMySQLStorage(db *sqlx.DB) (Storage, error) {
	return sql.NewSQLStorage(db, "mysql")
}

// NewPostgreSQLStorage creates a PostgreSQL storage backend with auto-migration
//
// Example:
//
//	db, _ := sqlx.Connect("postgres", "postgres://user:pass@localhost/db?sslmode=disable")
//	storage, _ := flow.NewPostgreSQLStorage(db)
//
// Tables are automatically created if they don't exist.
func NewPostgreSQLStorage(db *sqlx.DB) (Storage, error) {
	return sql.NewSQLStorage(db, "postgres")
}

// NewRedisStorage creates a Redis storage backend
//
// Example:
//
//	storage := flow.NewRedisStorage("localhost:6379", "", 0)
func NewRedisStorage(addr string, password string, db int) Storage {
	return redis.NewRedisStorage(addr, password, db)
}

// ========================================
// ENCRYPTION
// ========================================

// Crypto provides data-at-rest encryption
type Crypto = storage.Crypto

// NewCrypto creates a new encryption instance
//
// Example:
//
//	crypto, _ := flow.NewCrypto("your-32-byte-encryption-key-here")
//	sqlStorage.(*sql.SQLStorage).SetEncryption(crypto)
func NewCrypto(key string) (*Crypto, error) {
	return storage.NewCrypto(key)
}

// ========================================
// WORKFLOW BUILDER
// ========================================

// WorkflowBuilder provides a fluent API for building workflows
type WorkflowBuilder struct {
	*builder.WorkflowBuilder
}

// NodeBuilder provides a fluent API for configuring nodes
type NodeBuilder struct {
	*builder.NodeBuilder
}

// NewWorkflow creates a new workflow builder
//
// Example:
//
//	wf := flow.NewWorkflow("order-processing", "Order Processing Workflow").
//	    AddNode("validate", flow.NodeTypeFunction).
//	        WithConfig("functionName", "ValidateOrder").
//	        Then("process").
//	    AddNode("process", flow.NodeTypeFunction).
//	        WithConfig("functionName", "ProcessOrder").
//	    Build()
func NewWorkflow(id, name string) *WorkflowBuilder {
	return &WorkflowBuilder{builder.NewWorkflow(id, name)}
}

// AddNode adds a node to the workflow
func (b *WorkflowBuilder) AddNode(id, nodeType string) *NodeBuilder {
	return &NodeBuilder{b.WorkflowBuilder.AddNode(id, nodeType)}
}

// Connect creates an edge between two nodes
func (b *WorkflowBuilder) Connect(sourceID, targetID string) *WorkflowBuilder {
	b.WorkflowBuilder.Connect(sourceID, targetID)
	return b
}

// ConnectIf creates a conditional edge
func (b *WorkflowBuilder) ConnectIf(sourceID, targetID, condition string) *WorkflowBuilder {
	b.WorkflowBuilder.ConnectIf(sourceID, targetID, condition)
	return b
}

// Build returns the completed workflow
func (b *WorkflowBuilder) Build() *Workflow {
	return b.WorkflowBuilder.Build()
}

// Visualize generates a Mermaid diagram of the workflow
func (b *WorkflowBuilder) Visualize() string {
	return b.WorkflowBuilder.Visualise()
}

// WithConfig adds configuration to a node
func (nb *NodeBuilder) WithConfig(key string, value interface{}) *NodeBuilder {
	nb.NodeBuilder.WithConfig(key, value)
	return nb
}

// WithSaga sets a compensation node (saga pattern)
func (nb *NodeBuilder) WithSaga(compensateNodeID string) *NodeBuilder {
	nb.NodeBuilder.WithSaga(compensateNodeID)
	return nb
}

// WithRetry sets retry count
func (nb *NodeBuilder) WithRetry(count int) *NodeBuilder {
	nb.NodeBuilder.WithRetry(count)
	return nb
}

// Then connects this node to the next node
func (nb *NodeBuilder) Then(targetID string) *NodeBuilder {
	return &NodeBuilder{nb.NodeBuilder.Then(targetID)}
}

// If creates a conditional connection
func (nb *NodeBuilder) If(targetID, condition string) *NodeBuilder {
	return &NodeBuilder{nb.NodeBuilder.If(targetID, condition)}
}

// ToMermaid generates a Mermaid.js diagram from a workflow
func ToMermaid(wf *Workflow) string {
	return builder.ToMermaid(wf)
}

// ========================================
// DAG / GRAPH
// ========================================

// NewGraph creates a new graph from a workflow
func NewGraph(wf *Workflow) (*Graph, error) {
	return engine.NewGraph(wf)
}

// ========================================
// TRIGGERS
// ========================================

// CronManager manages scheduled workflow execution
type CronManager struct {
	*trigger.CronManager
}

// NewCronManager creates a new cron scheduler
//
// Example:
//
//	cronMgr := flow.NewCronManager(engine, logger)
//	cronMgr.Start()
//	cronMgr.AddSchedule("default", "backup-workflow", "0 2 * * *", `{"type":"daily"}`)
func NewCronManager(eng *Engine, logger *slog.Logger) *CronManager {
	return &CronManager{trigger.NewCronManager(eng.Engine, logger)}
}

// Start begins the cron scheduler
func (c *CronManager) Start() {
	c.CronManager.Start()
}

// Stop halts the cron scheduler
func (c *CronManager) Stop() {
	c.CronManager.Stop()
}

// AddSchedule schedules a workflow with cron expression
func (c *CronManager) AddSchedule(namespace, workflowID, cronExpr, input string) (cron.EntryID, error) {
	return c.CronManager.AddSchedule(namespace, workflowID, cronExpr, input)
}

// WebhookManager handles HTTP webhook triggers
type WebhookManager struct {
	*trigger.WebhookManager
}

// NewWebhookManager creates a webhook handler
//
// Example:
//
//	webhookMgr := flow.NewWebhookManager(engine)
//	http.Handle("/webhooks/", webhookMgr)
//
// POST /webhooks/{namespace}/{workflow_id} with JSON body to trigger workflows
func NewWebhookManager(eng *Engine) *WebhookManager {
	return &WebhookManager{trigger.NewWebhookManager(eng.Engine)}
}

// ServeHTTP handles webhook requests
func (w *WebhookManager) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	w.WebhookManager.ServeHTTP(rw, r)
}

// APIManager provides REST API for workflow management
type APIManager struct {
	*trigger.APIManager
}

// NewAPIManager creates a REST API handler
//
// Example:
//
//	apiMgr := flow.NewAPIManager(webhookMgr)
//	http.Handle("/", apiMgr)
//
// Endpoints:
//   - GET /health
//   - GET /metrics (Prometheus)
//   - GET /api/{namespace}/executions/{id}
//   - POST /api/{namespace}/executions/{id}/cancel
//   - POST /api/{namespace}/executions/{id}/signal/{nodeID}
func NewAPIManager(webhookMgr *WebhookManager) *APIManager {
	return &APIManager{trigger.NewAPIManager(webhookMgr.WebhookManager)}
}

// ServeHTTP handles API requests
func (a *APIManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.APIManager.ServeHTTP(w, r)
}

// ========================================
// EVENTS
// ========================================

// EventType represents workflow event types
type EventType = engine.EventType

// Event types
const (
	EventWorkflowStarted       = engine.EventTypeWorkflowStarted
	EventWorkflowCompleted     = engine.EventTypeWorkflowCompleted
	EventWorkflowFailed        = engine.EventTypeWorkflowFailed
	EventWorkflowCancelled     = engine.EventTypeWorkflowCancelled
	EventStepStarted           = engine.EventTypeStepStarted
	EventStepCompleted         = engine.EventTypeStepCompleted
	EventStepFailed            = engine.EventTypeStepFailed
	EventStepRetrying          = engine.EventTypeStepRetrying
	EventStepWaitingSignal     = engine.EventTypeStepWaitingSignal
	EventSignalReceived        = engine.EventTypeSignalReceived
	EventCompensationTriggered = engine.EventTypeCompensationTriggered
)

// Event represents a workflow event
type Event = engine.Event

// EventHandler handles workflow events
type EventHandler = engine.EventHandler

// EventBus manages event subscriptions
type EventBus struct {
	*engine.EventBus
}

// NewEventBus creates a new event bus
//
// Example:
//
//	eventBus := flow.NewEventBus()
//	eventBus.RegisterHandler(flow.EventWorkflowCompleted, myHandler)
func NewEventBus() *EventBus {
	return &EventBus{engine.NewEventBus()}
}

// RegisterHandler registers a handler for a specific event type
func (eb *EventBus) RegisterHandler(eventType EventType, handler EventHandler) {
	eb.EventBus.RegisterHandler(eventType, handler)
}

// RegisterGlobalHandler registers a handler for all events
func (eb *EventBus) RegisterGlobalHandler(handler EventHandler) {
	eb.EventBus.RegisterGlobalHandler(handler)
}

// Emit emits an event asynchronously
func (eb *EventBus) Emit(ctx context.Context, event Event) {
	eb.EventBus.Emit(ctx, event)
}

// EmitSync emits an event synchronously
func (eb *EventBus) EmitSync(ctx context.Context, event Event) []error {
	return eb.EventBus.EmitSync(ctx, event)
}

// ========================================
// TELEMETRY / METRICS
// ========================================

// Prometheus metrics (pre-registered)
var (
	WorkflowsStarted   = telemetry.WorkflowsStarted
	WorkflowsCompleted = telemetry.WorkflowsCompleted
	WorkflowsFailed    = telemetry.WorkflowsFailed
	StepsProcessed     = telemetry.StepsProcessed
	StepDuration       = telemetry.StepDuration
	ActiveWorkers      = telemetry.ActiveWorkers
)

// MetricsRegistry returns the Prometheus registry
func MetricsRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}

// ========================================
// BUILT-IN NODE TYPES
// ========================================

// Built-in node executors (auto-registered)
var (
	FunctionNode    = &node.FunctionNode{}
	HTTPNode        = &node.HTTPNode{}
	ConditionNode   = &node.ConditionNode{}
	TransformNode   = &node.TransformNode{}
	LogNode         = &node.LogNode{}
	SignalNode      = &node.SignalNode{}
	SubWorkflowNode = &node.SubWorkflowNode{}
	AINode          = &node.AINode{}
	NotifyNode      = &node.NotifyNode{}
	SwitchNode      = &node.SwitchNode{}
	WaitNode        = &node.WaitNode{}
	SetVariableNode = &node.SetVariableNode{}
)
