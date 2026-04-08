# Parevo Flow 🌊

**High-Performance Workflow Orchestration Engine for Go**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

Parevo Flow is a production-ready, DAG-based workflow orchestration engine that delivers **Temporal-level reliability** with **minimal complexity**. Built for developers who need durable execution, distributed task processing, and human-in-the-loop workflows—without the operational overhead.

---

## 🎯 Why Parevo Flow?

**Simple Yet Powerful**
- ✅ **Single Import** - `github.com/parevo/flow` - everything you need
- ✅ **Zero Setup** - Auto-creates database tables on first run
- ✅ **Embedded Library** - No separate servers or infrastructure
- ✅ **Multiple Storage** - MySQL, PostgreSQL, Redis, or in-memory

**Production-Ready**
- 🔄 **Auto-Retry** - Configurable retry policies with exponential backoff
- 🛡️ **Zombie Recovery** - Automatic detection and recovery of dead workers
- 🔐 **Encryption** - AES-256-GCM for sensitive data at rest
- 📊 **Observability** - Built-in Prometheus metrics and structured logging

**Developer-Friendly**
- 🎨 **Fluent API** - Intuitive workflow builder DSL
- 🚦 **Signals** - Human-in-the-loop and external event integration
- 🎭 **Built-in Nodes** - HTTP, AI, notifications, conditions, and more
- 📈 **Visual Builder** - Generate Mermaid diagrams from workflows

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/parevo/flow
```

**That's it!** All dependencies (MySQL driver, PostgreSQL driver, Redis client, etc.) are automatically installed.

### Complete Working Example (5 Minutes)

```go
package main

import (
    "context"
    "log"
    
    "github.com/parevo/flow"
    _ "github.com/go-sql-driver/mysql"  // MySQL driver (auto-installed)
    "github.com/jmoiron/sqlx"           // SQL extensions (auto-installed)
)

func main() {
    // 1. Connect to MySQL
    dsn := "user:pass@tcp(localhost:3306)/mydb?parseTime=true"
    db, err := sqlx.Connect("mysql", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // 2. Create storage (TABLES AUTO-CREATED! ✨)
    storage, err := flow.NewMySQLStorage(db)
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Setup engine
    registry := flow.NewRegistry()
    engine := flow.NewEngine(storage, registry)
    
    // 4. Register your functions
    registry.RegisterFunction("SendEmail", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
        email := input["email"].(string)
        log.Printf("📧 Sending email to: %s", email)
        // Your email logic here
        return map[string]interface{}{"sent": true}, nil
    })
    
    registry.RegisterFunction("CreateAccount", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
        username := input["username"].(string)
        log.Printf("✅ Account created: %s", username)
        return map[string]interface{}{"user_id": 12345}, nil
    })
    
    // 5. Define workflow
    wf := flow.NewWorkflow("user-onboarding", "User Onboarding").
        AddNode("create", flow.NodeTypeFunction).
            WithConfig("functionName", "CreateAccount").
            Then("email").
        AddNode("email", flow.NodeTypeFunction).
            WithConfig("functionName", "SendEmail").
        Build()
    
    // 6. Register workflow
    ctx := context.Background()
    engine.RegisterWorkflow(ctx, "default", wf)
    
    // 7. Start worker (processes tasks in background)
    go engine.StartWorker(ctx, "default", "worker-1")
    
    // 8. Execute workflow
    input := []byte(`{"username": "john", "email": "john@example.com"}`)
    execID, err := engine.Execute(ctx, "default", "user-onboarding", input)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("✅ Workflow started: %s", execID)
    
    // 9. Check status
    execution, _ := engine.GetExecution(ctx, "default", execID)
    log.Printf("Status: %s", execution.Status)
    
    // Keep running...
    select {}
}
```

**That's it!** Run this and you have a working workflow system with:
- ✅ Persistent storage (MySQL tables auto-created)
- ✅ Distributed workers (add more with different IDs)
- ✅ Automatic retries and error handling
- ✅ Full execution history

---

## 📦 Storage Backends

### MySQL (Production - Recommended)

```go
db, _ := sqlx.Connect("mysql", "user:pass@tcp(localhost:3306)/db?parseTime=true")
storage, _ := flow.NewMySQLStorage(db)
```

**Features:**
- ✅ Auto-creates tables and indexes on startup
- ✅ SKIP LOCKED for distributed workers
- ✅ Full ACID guarantees
- ✅ No manual schema setup required

**Tables Created Automatically:**
- `workflows` - Workflow definitions
- `executions` - Workflow runs
- `execution_steps` - Individual task states

### PostgreSQL (Production)

```go
db, _ := sqlx.Connect("postgres", "postgres://user:pass@localhost/db?sslmode=disable")
storage, _ := flow.NewPostgreSQLStorage(db)
```

Same features as MySQL with PostgreSQL-specific optimizations.

### Redis (High-Performance)

```go
storage := flow.NewRedisStorage("localhost:6379", "", 0)
```

**Best for:**
- Sub-millisecond task claims
- High-throughput scenarios
- Ephemeral workflows

### Memory (Development)

```go
storage := flow.NewMemoryStorage()
```

**Perfect for:**
- Unit tests
- Local development
- Prototyping

---

## 🏗️ Building Workflows

### Fluent Builder API

```go
wf := flow.NewWorkflow("order-processing", "Process Orders").
    // Node 1: Validate
    AddNode("validate", flow.NodeTypeFunction).
        WithConfig("functionName", "ValidateOrder").
        WithRetry(3).
        Then("check-stock").
    
    // Node 2: Check Stock
    AddNode("check-stock", flow.NodeTypeFunction).
        WithConfig("functionName", "CheckInventory").
        Then("charge").
    
    // Node 3: Charge Payment
    AddNode("charge", flow.NodeTypeFunction).
        WithConfig("functionName", "ChargeCard").
        WithSaga("refund").  // Compensation if fails
        Then("ship").
    
    // Node 4: Ship Order
    AddNode("ship", flow.NodeTypeFunction).
        WithConfig("functionName", "ShipOrder").
        Then("notify").
    
    // Node 5: Notify Customer
    AddNode("notify", flow.NodeTypeFunction).
        WithConfig("functionName", "SendNotification").
    
    // Compensation Node (Saga Pattern)
    AddNode("refund", flow.NodeTypeFunction).
        WithConfig("functionName", "RefundPayment").
    
    Build()

// Register and use
engine.RegisterWorkflow(ctx, "default", wf)
```

### Manual Definition

```go
wf := &flow.Workflow{
    ID:   "my-workflow",
    Name: "My Workflow",
    Nodes: []flow.Node{
        {
            ID:   "task1",
            Type: flow.NodeTypeFunction,
            Config: map[string]interface{}{
                "functionName": "MyFunction",
            },
        },
        {
            ID:   "task2",
            Type: flow.NodeTypeHTTP,
            Config: map[string]interface{}{
                "url":    "https://api.example.com/webhook",
                "method": "POST",
            },
        },
    },
    Edges: []flow.Edge{
        {SourceID: "task1", TargetID: "task2"},
    },
}
```

---

## 🎭 Built-in Node Types

### Function Node

Execute custom Go functions.

```go
registry.RegisterFunction("ProcessOrder", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    orderID := input["order_id"].(string)
    // Your business logic
    return map[string]interface{}{"status": "processed"}, nil
})
```

```go
.AddNode("process", flow.NodeTypeFunction).
    WithConfig("functionName", "ProcessOrder")
```

### HTTP Node

Make HTTP requests.

```go
.AddNode("webhook", flow.NodeTypeHTTP).
    WithConfig("url", "https://api.example.com/notify").
    WithConfig("method", "POST").
    WithConfig("body", `{"event": "order.completed"}`)
```

### Condition Node

Branch based on conditions.

```go
.AddNode("check", flow.NodeTypeCondition).
    WithConfig("expression", "input.amount > 1000")

// Connect with conditions
.ConnectIf("check", "approve", "true").
.ConnectIf("check", "auto-process", "false")
```

### Signal Node (Human-in-the-Loop)

Wait for external signals.

```go
.AddNode("await-approval", flow.NodeTypeSignal).
    WithConfig("signalName", "manager_approval")
```

Resume later:
```go
engine.SignalExecution(ctx, "default", execID, "await-approval", `{"approved": true}`)
```

### Transform Node

Transform data with Go templates.

```go
.AddNode("format", flow.NodeTypeTransform).
    WithConfig("template", `{"fullName": "{{.firstName}} {{.lastName}}"}`)
```

### AI Node

Call LLM APIs (OpenAI, Anthropic).

```go
.AddNode("summarize", flow.NodeTypeAI).
    WithConfig("provider", "openai").
    WithConfig("model", "gpt-4").
    WithConfig("prompt", "Summarize this order: {{.orderDetails}}")
```

### More Built-in Nodes

- `NodeTypeLog` - Structured logging
- `NodeTypeNotify` - Slack/Discord/Email notifications
- `NodeTypeSubWorkflow` - Nested workflows
- `NodeTypeSwitch` - Multi-way branching
- `NodeTypeWait` - Delay execution
- `NodeTypeSetVariable` - Context manipulation

---

## 🔄 Worker Management

### Single Worker

```go
go engine.StartWorker(ctx, "default", "worker-1")
```

### Multiple Workers (Horizontal Scaling)

```go
for i := 1; i <= 5; i++ {
    workerID := fmt.Sprintf("worker-%d", i)
    go engine.StartWorker(ctx, "default", workerID)
}
```

Workers automatically:
- ✅ Claim tasks using database locks
- ✅ Recover from crashes (zombie detection)
- ✅ Retry failed tasks
- ✅ Update metrics

### Separate Worker Process

**main.go** (API server)
```go
// Just start workflows, don't process
execID, _ := engine.Execute(ctx, "default", "my-workflow", input)
```

**worker.go** (Task processor)
```go
func main() {
    // Setup storage and engine...
    
    // Start worker
    engine.StartWorker(context.Background(), "default", os.Getenv("WORKER_ID"))
}
```

---

## 🚦 Advanced Features

### Retry Policies

```go
wf.Nodes = []flow.Node{
    {
        ID:   "flaky-task",
        Type: flow.NodeTypeFunction,
        Config: map[string]interface{}{
            "functionName": "FlakyAPI",
        },
        RetryPolicy: &flow.RetryPolicy{
            MaxAttempts:       5,
            InitialInterval:   time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:   time.Minute,
        },
    },
}
```

### Saga Pattern (Compensation)

```go
.AddNode("charge", flow.NodeTypeFunction).
    WithConfig("functionName", "ChargeCard").
    WithSaga("refund").  // Runs on failure

.AddNode("refund", flow.NodeTypeFunction).
    WithConfig("functionName", "RefundPayment")
```

### Cron Scheduling

```go
cronMgr := flow.NewCronManager(engine, logger)
cronMgr.Start()

// Run daily at 2 AM
cronMgr.AddSchedule("default", "daily-backup", "0 2 * * *", `{"type":"full"}`)

// Run every 5 minutes
cronMgr.AddSchedule("default", "health-check", "*/5 * * * *", `{}`)
```

### Webhooks

```go
webhookMgr := flow.NewWebhookManager(engine)
http.Handle("/webhooks/", webhookMgr)
go http.ListenAndServe(":8080", nil)

// Trigger with:
// POST /webhooks/default/my-workflow
// Body: {"key": "value"}
```

### REST API

```go
apiMgr := flow.NewAPIManager(webhookMgr)
http.Handle("/", apiMgr)

// Endpoints:
// GET  /health
// GET  /metrics (Prometheus)
// GET  /api/{namespace}/executions/{id}
// POST /api/{namespace}/executions/{id}/cancel
// POST /api/{namespace}/executions/{id}/signal/{nodeID}
```

### Event Bus

```go
eventBus := flow.NewEventBus()

// Register handler
eventBus.RegisterHandler(flow.EventWorkflowCompleted, 
    HandlerFunc(func(ctx context.Context, event flow.Event) error {
        log.Printf("Workflow completed: %s", event.ExecutionID)
        return nil
    }))

// Attach to engine
engine.SetEventBus(eventBus)
```

### Encryption (Data at Rest)

```go
crypto, _ := flow.NewCrypto("your-32-byte-encryption-key!!")

// For SQL storage
sqlStorage, _ := flow.NewMySQLStorage(db)
sqlStorage.(*sql.SQLStorage).SetEncryption(crypto)

// Input/output/error fields are now encrypted in database
```

---

## 📊 Monitoring

### Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

http.Handle("/metrics", promhttp.Handler())
```

**Available Metrics:**
- `workflows_started_total` - Counter of started workflows
- `workflows_completed_total` - Counter of completed workflows
- `workflows_failed_total` - Counter of failed workflows
- `steps_processed_total` - Counter of processed steps
- `step_duration_seconds` - Histogram of step durations
- `active_workers` - Gauge of active workers

### Logging

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
engine.WithLogger(logger)
```

### Visual Workflow Diagrams

```go
mermaidDiagram := flow.ToMermaid(wf)
fmt.Println(mermaidDiagram)
```

Output:
```mermaid
flowchart TD
    validate[Validate Order]
    charge[Charge Payment]
    ship[Ship Order]
    validate --> charge
    charge --> ship
```

---

## 🔍 Querying Workflows

### Get Execution Status

```go
execution, err := engine.GetExecution(ctx, "default", execID)
fmt.Println(execution.Status) // PENDING, RUNNING, COMPLETED, FAILED
```

### Get Execution Steps

```go
steps, err := engine.GetExecutionSteps(ctx, "default", execID)
for _, step := range steps {
    fmt.Printf("%s: %s\n", step.NodeID, step.Status)
}
```

### Cancel Execution

```go
err := engine.CancelExecution(ctx, "default", execID)
```

### Query Database Directly (MySQL)

```sql
-- Recent executions
SELECT id, workflow_id, status, started_at, finished_at
FROM executions
WHERE namespace = 'default'
ORDER BY started_at DESC
LIMIT 10;

-- Execution steps
SELECT node_id, status, started_at, finished_at, error
FROM execution_steps
WHERE execution_id = 'exec-uuid-xxx'
ORDER BY started_at;

-- Failed workflows
SELECT workflow_id, COUNT(*) as failures
FROM executions
WHERE status = 'FAILED'
  AND started_at > NOW() - INTERVAL 24 HOUR
GROUP BY workflow_id;
```

---

## 🏭 Production Deployment

### Docker Compose

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: parevo_flow
      MYSQL_USER: flowuser
      MYSQL_PASSWORD: flowpass
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"

  api:
    build: .
    environment:
      MYSQL_DSN: "flowuser:flowpass@tcp(mysql:3306)/parevo_flow?parseTime=true"
      WORKER_ENABLED: "false"
    ports:
      - "8080:8080"
    depends_on:
      - mysql

  worker:
    build: .
    environment:
      MYSQL_DSN: "flowuser:flowpass@tcp(mysql:3306)/parevo_flow?parseTime=true"
      WORKER_ENABLED: "true"
      WORKER_ID: "${HOSTNAME}"
    deploy:
      replicas: 3
    depends_on:
      - mysql

volumes:
  mysql_data:
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: parevo-worker
spec:
  replicas: 5
  selector:
    matchLabels:
      app: parevo-worker
  template:
    metadata:
      labels:
        app: parevo-worker
    spec:
      containers:
      - name: worker
        image: mycompany/parevo-app:latest
        env:
        - name: MYSQL_DSN
          valueFrom:
            secretKeyRef:
              name: mysql-credentials
              key: dsn
        - name: WORKER_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

### Environment Variables

```bash
# Database
MYSQL_DSN="user:pass@tcp(host:3306)/db?parseTime=true"

# Worker
WORKER_ENABLED="true"
WORKER_ID="worker-1"
WORKER_CONCURRENCY="10"

# Encryption (optional)
ENCRYPTION_KEY="your-32-byte-key-goes-here!!"

# Logging
LOG_LEVEL="info"
LOG_FORMAT="json"
```

---

## 🆚 Comparison

### vs. Temporal/Cadence

| Feature | Parevo Flow | Temporal |
|---------|-------------|----------|
| **Setup** | Embedded library | Separate server cluster |
| **Complexity** | Low | High |
| **Storage** | MySQL/Postgres/Redis | Cassandra/PostgreSQL |
| **Language** | Go | Multi-language |
| **Use Case** | Embedded workflows | Enterprise-scale |

**Choose Parevo Flow if:**
- You want embedded workflow engine
- You have existing MySQL/PostgreSQL
- You need quick setup and deployment
- Your team knows Go

**Choose Temporal if:**
- You need multi-language support
- You have dedicated infrastructure team
- You need massive scale (1000+ workflows/sec)

### vs. Asynq/Machinery (Job Queues)

| Feature | Parevo Flow | Asynq |
|---------|-------------|-------|
| **Workflows** | ✅ DAG, branching | ❌ Single tasks |
| **Dependencies** | ✅ Built-in | ❌ Manual |
| **Retries** | ✅ Per-node policies | ✅ Global policy |
| **Signals** | ✅ Human-in-the-loop | ❌ |
| **Compensation** | ✅ Saga pattern | ❌ |

**Choose Parevo Flow for:**
- Multi-step business processes
- Conditional logic
- Human approvals
- Long-running workflows

**Choose Asynq for:**
- Simple background jobs
- Pure task queue
- Redis-only architecture

---

## 📚 Documentation

- **[API Reference](https://pkg.go.dev/github.com/parevo/flow)** - Full API documentation
- **[Examples](./examples/)** - Working code examples
- **[Visual Builder](./examples/visual_builder/)** - Interactive workflow builder

---

## 🧪 Testing

```go
func TestMyWorkflow(t *testing.T) {
    // Use in-memory storage for tests
    storage := flow.NewMemoryStorage()
    registry := flow.NewRegistry()
    engine := flow.NewEngine(storage, registry)
    
    // Register test functions
    registry.RegisterFunction("TestFunc", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
        return map[string]interface{}{"result": "success"}, nil
    })
    
    // Create and register workflow
    wf := flow.NewWorkflow("test-wf", "Test").
        AddNode("task", flow.NodeTypeFunction).
            WithConfig("functionName", "TestFunc").
        Build()
    
    ctx := context.Background()
    engine.RegisterWorkflow(ctx, "test", wf)
    
    // Execute
    execID, err := engine.Execute(ctx, "test", "test-wf", []byte(`{}`))
    assert.NoError(t, err)
    
    // Verify
    exec, _ := engine.GetExecution(ctx, "test", execID)
    assert.Equal(t, flow.StatusCompleted, exec.Status)
}
```

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Areas we'd love help with:**
- Additional storage backends (MongoDB, DynamoDB)
- More built-in node types
- Performance optimizations
- Documentation improvements

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 💬 Support

- **Issues**: [GitHub Issues](https://github.com/parevo/flow/issues)
- **Discussions**: [GitHub Discussions](https://github.com/parevo/flow/discussions)
- **Email**: support@parevo.io

---

## 🙏 Acknowledgments

Inspired by:
- **Temporal.io** - Durable execution patterns
- **Cadence** - Workflow orchestration concepts
- **Asynq** - Task queue simplicity

Built with ❤️ by the Parevo team.

---

## 🗺️ Roadmap

- [ ] **v1.1** - WebAssembly node support
- [ ] **v1.2** - GraphQL API
- [ ] **v1.3** - UI dashboard
- [ ] **v2.0** - Multi-region replication
- [ ] **v2.1** - Workflow versioning and rollback

---

**⭐ Star us on GitHub if you find Parevo Flow useful!**
