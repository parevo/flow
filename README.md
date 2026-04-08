# Parevo Flow

A workflow orchestration engine for Go applications. Provides DAG-based execution, distributed task processing, and durable state management.

## Installation

```bash
go get github.com/parevo/flow
```

All dependencies (database drivers, Redis client, etc.) are automatically installed.

## Requirements

- Go 1.23 or higher
- MySQL 5.7+, PostgreSQL 12+, or Redis 6+ (optional - in-memory storage available)

## Quick Start

```go
package main

import (
    "context"
    "github.com/parevo/flow"
    _ "github.com/go-sql-driver/mysql"
    "github.com/jmoiron/sqlx"
)

func main() {
    // Initialize storage
    db, _ := sqlx.Connect("mysql", "user:pass@tcp(localhost:3306)/db?parseTime=true")
    storage, _ := flow.NewMySQLStorage(db)
    
    // Create engine and registry
    registry := flow.NewRegistry()
    engine := flow.NewEngine(storage, registry)
    
    // Register function
    registry.RegisterFunction("ProcessData", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
        // Your logic here
        return map[string]interface{}{"status": "processed"}, nil
    })
    
    // Define workflow
    wf := flow.NewWorkflow("process-workflow", "Data Processing").
        AddNode("process", flow.NodeTypeFunction).
            WithConfig("function", "ProcessData").
        Build()
    
    // Register and execute
    ctx := context.Background()
    engine.RegisterWorkflow(ctx, "default", wf)
    
    // Start worker
    go engine.StartWorker(ctx, "default", "worker-1")
    
    // Execute workflow
    execID, _ := engine.Execute(ctx, "default", "process-workflow", []byte(`{"data":"value"}`))
    
    // Query status
    execution, _ := engine.GetExecution(ctx, "default", execID)
}
```

## Storage Backends

### MySQL

```go
db, _ := sqlx.Connect("mysql", "user:pass@tcp(host:3306)/db?parseTime=true")
storage, _ := flow.NewMySQLStorage(db)
```

Database schema is created automatically on first connection.

### PostgreSQL

```go
db, _ := sqlx.Connect("postgres", "postgres://user:pass@host/db?sslmode=disable")
storage, _ := flow.NewPostgreSQLStorage(db)
```

### Redis

```go
storage := flow.NewRedisStorage("localhost:6379", "", 0)
```

### In-Memory

```go
storage := flow.NewMemoryStorage()
```

Suitable for development and testing.

## API Reference

### Engine

```go
engine := flow.NewEngine(storage, registry)
engine.WithLogger(logger)
engine.RegisterWorkflow(ctx, namespace, workflow)
engine.Execute(ctx, namespace, workflowID, input)
engine.GetExecution(ctx, namespace, executionID)
engine.GetExecutionSteps(ctx, namespace, executionID)
engine.CancelExecution(ctx, namespace, executionID)
engine.FailExecution(ctx, namespace, executionID, message)
engine.SignalExecution(ctx, namespace, executionID, nodeID, data)
engine.StartWorker(ctx, namespace, workerID)
```

### Registry

```go
registry := flow.NewRegistry()
registry.RegisterFunction(name, handlerFunc)
registry.Register(nodeType, executor)
```

### Workflow Builder

```go
wf := flow.NewWorkflow(id, name).
    AddNode(nodeID, nodeType).
        WithConfig(key, value).
        WithRetry(count).
        WithSaga(compensateNodeID).
    Connect(sourceID, targetID).
    ConnectIf(sourceID, targetID, condition).
    Build()
```

### Worker

```go
worker := flow.NewWorker(workerID, engine, registry, interval)
worker.SetNamespace(namespace)
worker.Start(ctx)
```

## Node Types

Built-in node types are automatically registered:

- `function` - Execute registered Go functions
- `http` - HTTP requests
- `condition` - Conditional branching
- `log` - Logging
- `transform` - Data transformation
- `signal` - Wait for external signals
- `subworkflow` - Nested workflows
- `ai` - LLM API calls
- `notify` - Notifications
- `switch` - Multi-way branching
- `wait` - Delay execution
- `setvariable` - Context manipulation

## Configuration

### Retry Policy

```go
node := flow.Node{
    ID: "task",
    Type: flow.NodeTypeFunction,
    RetryPolicy: &flow.RetryPolicy{
        MaxAttempts:        5,
        InitialInterval:    time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval:    time.Minute,
    },
}
```

### Compensation (Saga Pattern)

```go
builder.AddNode("charge", flow.NodeTypeFunction).
    WithSaga("refund")
```

### Encryption

```go
crypto, _ := flow.NewCrypto("your-32-byte-encryption-key-here")
storage.(*sql.SQLStorage).SetEncryption(crypto)
```

## Triggers

### Cron

```go
cronMgr := flow.NewCronManager(engine, logger)
cronMgr.Start()
cronMgr.AddSchedule(namespace, workflowID, "0 2 * * *", input)
```

### Webhooks

```go
webhookMgr := flow.NewWebhookManager(engine)
http.Handle("/webhooks/", webhookMgr)
```

### REST API

```go
apiMgr := flow.NewAPIManager(webhookMgr)
http.Handle("/", apiMgr)
```

Endpoints:
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics
- `GET /api/{namespace}/executions/{id}` - Get execution
- `POST /api/{namespace}/executions/{id}/cancel` - Cancel execution
- `POST /api/{namespace}/executions/{id}/signal/{nodeID}` - Send signal

## Events

```go
eventBus := flow.NewEventBus()
eventBus.RegisterHandler(flow.EventWorkflowCompleted, handler)
eventBus.RegisterGlobalHandler(handler)
```

Event types:
- `EventWorkflowStarted`
- `EventWorkflowCompleted`
- `EventWorkflowFailed`
- `EventStepStarted`
- `EventStepCompleted`
- `EventStepFailed`

## Monitoring

### Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

http.Handle("/metrics", promhttp.Handler())
```

Available metrics:
- `workflows_started_total`
- `workflows_completed_total`
- `workflows_failed_total`
- `steps_processed_total`
- `step_duration_seconds`
- `active_workers`

### Logging

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
engine.WithLogger(logger)
```

## Testing

```bash
# Run tests
go test

# Run with coverage
go test -cover

# Run with race detector
go test -race

# Run benchmark
go test -bench=.
```

## Examples

See `examples/` directory for complete working examples:
- `examples/visual_builder/` - Web-based workflow builder with drag-and-drop interface

## Database Schema

Tables are created automatically when using SQL storage backends:

- `workflows` - Workflow definitions
- `executions` - Workflow execution instances
- `execution_steps` - Individual task states

Indexes are optimized for:
- Namespace-based queries
- Status filtering
- Worker task claims
- Execution lookups

## Concurrency Model

- Workers claim tasks using database-level locking (`SKIP LOCKED`)
- Multiple workers can run concurrently across different processes/hosts
- Tasks are automatically reassigned if a worker crashes (zombie detection)
- No message broker required - coordination through storage backend

## Error Handling

- Failed tasks can be retried with configurable policies
- Saga pattern supported for compensation logic
- Workflow status tracked: `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`
- Task status tracked: `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, `SKIPPED`, `WAITING`, `CANCELLED`

## License

MIT License. See [LICENSE](LICENSE) file.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Documentation

Full API documentation: https://pkg.go.dev/github.com/parevo/flow
