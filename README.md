# Parevo Flow 🌊

### High-Performance, Minimalist Workflow Orchestration Engine for Go

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)
[![Performance](https://img.shields.io/badge/Performance-High-orange?style=for-the-badge)]()
[![Architecture](https://img.shields.io/badge/Architecture-DAG--Based-blue?style=for-the-badge)]()

**Parevo Flow** is a production-ready, DAG-based workflow orchestration engine that delivers **Temporal-level reliability** with **minimal complexity**. Built for developers who need durable execution, human-in-the-loop workflows, and distributed task processing—without the operational overhead.

---

## 🎯 Why Parevo Flow?

Modern applications need reliable workflow orchestration, but existing solutions are often **too heavy** (Temporal), **too simple** (job queues), or **too opaque** (cloud-specific). Parevo Flow hits the sweet spot:

- ✅ **Simple**: Embedded Go library, not a separate service
- ✅ **Powerful**: DAG execution, signals, child workflows, compensation
- ✅ **Reliable**: Durable execution, automatic retry, zombie recovery
- ✅ **Fast**: Redis-backed for high throughput, SQL for durability
- ✅ **Observable**: Prometheus metrics, structured logging, visual builder
- ✅ **Secure**: AES-256-GCM encryption for sensitive data

---

## ⚡ Quick Start

### Installation

```bash
go get github.com/parevo/flow
```

### Hello World (30 seconds)

```go
package main

import (
    "context"
    "github.com/parevo/flow/internal/builder"
    "github.com/parevo/flow/internal/engine"
    "github.com/parevo/flow/internal/node"
    "github.com/parevo/flow/internal/storage/memory"
)

func main() {
    // 1. Create storage
    storage := memory.NewMemoryStorage()
    
    // 2. Register nodes
    registry := engine.NewRegistry()
    registry.Register("log", &node.LogNode{})
    
    // 3. Create engine
    eng := engine.NewEngine(storage, registry)
    
    // 4. Define workflow
    wf := builder.NewWorkflowBuilder()
    wf.SetID("hello-world")
    wf.AddNode("greet", "log").WithConfig("message", "Hello, Parevo Flow!")
    workflow, _ := wf.Build()
    
    // 5. Save and execute
    storage.SaveWorkflow(context.Background(), "default", workflow)
    execID, _ := eng.StartWorkflow(context.Background(), "default", "hello-world", "{}")
    
    println("Execution ID:", execID)
}
```

### Real-World Example (2 minutes)

See **complete working examples** with Calendar Approval, Document Sharing, and Invoice Workflows:

👉 **[examples/integration/rest_api_chi.go](examples/integration/rest_api_chi.go)** - Full REST API with 3 business workflows  
👉 **[examples/integration/README.md](examples/integration/README.md)** - Detailed usage guide

```bash
# Run the example server
cd examples/integration
go run rest_api_chi.go

# Test workflows
./test_workflows.sh
```

---

## 🏗️ Core Features

### 1. 🔀 DAG-Based Execution

Build complex workflows as **Directed Acyclic Graphs** with parallel execution, conditional branching, and intelligent routing.

```go
wf := builder.NewWorkflowBuilder()
wf.SetID("user-onboarding")

// Parallel steps
wf.AddNode("send-email", "http").WithConfig("url", "https://api.email.com")
wf.AddNode("create-account", "function").WithConfig("function", "create_user")
wf.AddNode("setup-billing", "function").WithConfig("function", "setup_stripe")

// Conditional logic
wf.AddNode("check-plan", "condition").WithConfig("condition", "$.plan == 'premium'")
wf.AddNode("enable-features", "function").WithConfig("function", "premium_features")

// Connect nodes
wf.AddTransition("send-email", "create-account", "default")
wf.AddTransition("create-account", "check-plan", "default")
wf.AddTransition("check-plan", "enable-features", "true")
wf.AddTransition("check-plan", "setup-billing", "false")
```

### 2. 🎨 Fluent Builder DSL

Type-safe, chainable API for defining workflows in pure Go—no YAML, no JSON, no external DSL.

```go
wf.AddNode("step1", "http").
    WithConfig("url", "https://api.example.com").
    WithConfig("method", "POST").
    Then("step2").
    AddNode("step2", "transform").
    WithConfig("expression", "$.data.id").
    Then("step3")
```

### 3. 🚦 Human-in-the-Loop (Signals)

Pause workflows and wait for external input—perfect for approvals, reviews, and interactive processes.

```go
// Define signal node
wf.AddNode("wait-approval", "signal").
    WithConfig("timeout", 86400) // 24 hours

// Resume from API
POST /api/v1/workflows/{exec_id}/signal?node=wait-approval
{
  "data": {
    "approved": true,
    "approver": "manager@example.com"
  }
}
```

**Real Use Cases:**
- 📧 Email approval links
- 💬 Slack interactive buttons
- 📱 Mobile app confirmations
- 🌐 Web dashboard actions

### 4. 🔧 FunctionNode Pattern

Register custom business logic without creating node types—maximum flexibility, minimal code.

```go
funcNode := node.NewFunctionNode()

// Register business functions
funcNode.Register("send-invoice", func(ctx context.Context, config map[string]interface{}, input string) (string, error) {
    var data map[string]interface{}
    json.Unmarshal([]byte(input), &data)
    
    // Your business logic
    invoiceID := data["invoice_id"]
    err := deps.InvoiceService.Send(invoiceID)
    
    result := map[string]interface{}{"sent": true, "invoice_id": invoiceID}
    return json.Marshal(result)
})

registry.Register("function", funcNode)

// Use in workflow
wf.AddNode("send", "function").WithConfig("function", "send-invoice")
```

### 5. 🎭 Built-in Node Library

**8 production-ready node types** for common workflow patterns:

| Node Type | Purpose | Example Use Case |
|-----------|---------|------------------|
| **`condition`** | If/else branching | Route based on amount, status, plan type |
| **`switch`** | Multi-way routing | Route to different handlers by category |
| **`wait`** | Delay execution | Send reminder after 3 days |
| **`http`** | REST API calls | Call external services, webhooks |
| **`signal`** | Wait for external input | Human approvals, user confirmations |
| **`transform`** | Data manipulation | Extract fields, format data |
| **`subworkflow`** | Call child workflows | Modular, reusable workflow components |
| **`function`** | Custom business logic | Any custom code you need |

### 6. 🔁 Child Workflows (SubWorkflow)

Compose complex workflows from reusable components—DRY principle for workflows.

```go
// Parent workflow
wf.AddNode("process-order", "subworkflow").
    WithConfig("workflow_id", "payment-processing").
    WithConfig("namespace", "default")

// Child workflow (payment-processing) executes independently
// Parent workflow waits for child completion
```

### 7. 💾 Multiple Storage Backends

Choose the right storage for your needs—all with the same API.

#### Memory (Development)
```go
storage := memory.NewMemoryStorage()
```

#### SQL (Production - Durability)
```go
storage := sql.NewSQLStorage(db, "postgres")
// Supports: PostgreSQL, MySQL, SQLite
// Features: SKIP LOCKED for distributed workers
```

#### Redis (Production - Performance)
```go
storage := redis.NewRedisStorage("localhost:6379", "", 0)
// Features: Sub-millisecond operations, high throughput
// Perfect for: High-concurrency, low-latency requirements
```

### 8. 🛡️ Self-Healing & Fault Tolerance

**Automatic recovery** from worker crashes, network failures, and transient errors.

#### Zombie Task Recovery
Tasks stuck because of worker crash? Automatically reassigned.
```go
// Task not completed within visibility timeout?
// → Automatically released back to queue
// → Another worker picks it up
```

#### Automatic Retry
```go
wf.AddNode("api-call", "http").
    WithConfig("url", "https://flaky-api.com").
    WithConfig("retry", 3).
    WithConfig("retry_delay", 5000) // milliseconds
```

#### Compensation (Saga Pattern)
```go
wf.AddNode("payment", "function").
    WithConfig("function", "charge-card").
    WithSaga("refund-payment") // Runs if downstream fails
```

### 9. 📊 Visual Workflow Builder

Generate beautiful Mermaid.js diagrams from your Go code—perfect for documentation.

```go
mermaidDiagram := visualizer.GenerateMermaid(workflow)
fmt.Println(mermaidDiagram)
```

**Output:**
```mermaid
flowchart TD
    start[Start]
    process[Process Data]
    check{Check Status}
    success[Success Path]
    failure[Failure Path]
    
    start --> process
    process --> check
    check -->|success| success
    check -->|failure| failure
```

Paste directly into GitHub, Notion, Confluence, or [Mermaid Live](https://mermaid.live).

### 10. 📈 Observability & Monitoring

#### Prometheus Metrics (Built-in)
```go
// Automatic metrics
parevo_workflows_started_total
parevo_workflows_completed_total
parevo_workflows_failed_total
parevo_workflow_duration_seconds
parevo_node_execution_duration_seconds
```

#### Structured Logging
```go
eng := engine.NewEngine(storage, registry).
    WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

// Outputs structured logs
{"level":"info","msg":"workflow started","execution_id":"abc123","workflow_id":"user-onboarding"}
```

#### Event System
```go
// Emit domain events
eventBus.Emit(Event{
    Type: "workflow.completed",
    Data: map[string]interface{}{"execution_id": execID},
})

// Subscribe to events
eventBus.Subscribe("workflow.completed", func(e Event) error {
    // Send notification, update dashboard, etc.
})
```

### 11. 🔐 Enterprise Security

#### Data-at-Rest Encryption
```go
storage := sql.NewSQLStorage(db, "postgres").
    WithEncryption("your-32-byte-encryption-key-here")

// Sensitive data automatically encrypted with AES-256-GCM
// Transparent to your application code
```

### 12. 🚀 Distributed Execution

Run multiple worker processes for horizontal scaling.

```go
// Worker 1 (Server A)
worker1 := engine.NewWorker(storage, registry, "worker-1").
    WithConcurrency(10)
worker1.Start()

// Worker 2 (Server B)
worker2 := engine.NewWorker(storage, registry, "worker-2").
    WithConcurrency(10)
worker2.Start()

// Tasks automatically distributed
// Built-in coordination via storage backend
```

### 13. ⏰ Scheduled Workflows (Cron)

Trigger workflows periodically—no external scheduler needed.

```go
cronMgr := trigger.NewCronManager(engine, logger)

// Daily report at 9 AM
cronMgr.AddSchedule("default", "daily-report", "0 9 * * *", `{"date":"today"}`)

// Every 5 minutes
cronMgr.AddSchedule("default", "health-check", "*/5 * * * *", `{}`)

cronMgr.Start()
```

---

## 📦 What's Included

### Core Package Structure

```
workflow/
├── internal/
│   ├── engine/          # Workflow execution engine
│   │   ├── engine.go    # Main orchestration
│   │   ├── worker.go    # Distributed worker
│   │   ├── events.go    # Event system
│   │   └── telemetry.go # Metrics
│   ├── builder/         # Fluent DSL
│   │   ├── builder.go   # Workflow builder
│   │   └── visualizer.go # Mermaid generation
│   ├── node/            # Built-in nodes
│   │   ├── condition.go
│   │   ├── http.go
│   │   ├── signal.go
│   │   ├── function.go  # Custom logic
│   │   └── ...
│   ├── storage/         # Storage backends
│   │   ├── memory/      # In-memory
│   │   ├── sql/         # PostgreSQL/MySQL
│   │   └── redis/       # Redis
│   └── trigger/         # Workflow triggers
│       ├── webhook.go
│       └── cron.go
└── examples/
    └── integration/     # Complete working examples
        ├── rest_api_chi.go      # REST API server
        ├── test_workflows.sh    # Test script
        └── README.md            # Usage guide
```

---

## 🎯 Real-World Use Cases

### ✅ Implemented Examples (Ready to Use)

#### 1. Calendar Event Approval Workflow
```
Create Calendar → Send Invitations → Wait for Approval → Process Response → Notify
```
**Use case:** Meeting scheduling with manager approval, calendar invites with .ics attachments

#### 2. Document Sharing Workflow
```
Collect Documents → Wait for Review → Approve/Reject Files → Copy to Customer → Notify
```
**Use case:** Document review and approval system, file sharing workflows

#### 3. Invoice Approval Workflow
```
Check Amount → [Low: Auto-Approve | High: Wait Manager] → Finalize → Notify
```
**Use case:** Conditional approval based on business rules, expense management

### 💡 Perfect For

- 🏢 **SaaS Onboarding** - Multi-step user activation with email verification
- 💳 **Payment Processing** - Charge → Verify → Fulfill → Notify (with rollback)
- 📊 **Data Pipelines** - ETL workflows with transformation and validation
- 🔄 **Order Fulfillment** - Inventory → Payment → Shipping → Tracking
- 📧 **Email Campaigns** - Send → Wait → Track Opens → Follow-up
- 🤖 **AI Workflows** - Prompt → Process → Review → Publish
- 🏗️ **Infrastructure Automation** - Provision → Configure → Test → Deploy

---

## 🆚 Comparison with Alternatives

### Parevo Flow vs. Temporal/Cadence

| Feature | Temporal | Parevo Flow | Winner |
|---------|----------|-------------|--------|
| **Deployment** | Separate service cluster | Embedded library | ✅ Parevo |
| **Setup Time** | Hours/Days | Minutes | ✅ Parevo |
| **Binary Size** | ~100MB+ infrastructure | ~2MB library | ✅ Parevo |
| **Learning Curve** | Steep (Event Sourcing) | Gentle (State Machine) | ✅ Parevo |
| **Durability** | Event Replay | Direct State Persistence | 🤝 Tie |
| **Scalability** | Excellent | Excellent | 🤝 Tie |
| **Maturity** | Very High | Growing | ✅ Temporal |
| **Community** | Large | Small | ✅ Temporal |

**When to choose Parevo Flow:**
- ✅ You want embedded execution (no separate service)
- ✅ You need fast iteration and simple deployment
- ✅ You prefer explicit state management over event sourcing
- ✅ You want built-in REST API examples

**When to choose Temporal:**
- ✅ You need battle-tested, proven-at-scale solution
- ✅ You have complex polyglot requirements
- ✅ You have ops team to manage infrastructure

### Parevo Flow vs. Asynq/Machinery (Job Queues)

| Feature | Job Queues | Parevo Flow | Winner |
|---------|-----------|-------------|--------|
| **DAG Execution** | ❌ Linear only | ✅ Full DAG | ✅ Parevo |
| **Conditional Logic** | ❌ Manual | ✅ Built-in | ✅ Parevo |
| **Signals/Approvals** | ❌ No | ✅ Native | ✅ Parevo |
| **Visual Builder** | ❌ No | ✅ Yes | ✅ Parevo |
| **Simplicity** | ✅ Very simple | ⚠️ More complex | ✅ Job Queues |

**Use job queues for:** Simple background jobs, fire-and-forget tasks  
**Use Parevo Flow for:** Multi-step workflows, human-in-loop, complex routing

---

## 🚀 Production Deployment

### Docker Example

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o workflow-server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/workflow-server /usr/local/bin/
EXPOSE 8080
CMD ["workflow-server"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: parevo-flow-worker
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: worker
        image: your-app:latest
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
        - name: WORKER_CONCURRENCY
          value: "10"
```

### Environment Variables

```bash
# Storage
DATABASE_URL=postgres://user:pass@localhost:5432/workflows
REDIS_URL=redis://localhost:6379

# Worker
WORKER_CONCURRENCY=10
WORKER_ID=worker-1

# Monitoring
METRICS_PORT=9090
LOG_LEVEL=info

# Security
ENCRYPTION_KEY=your-32-byte-key-here
```

---

## 📚 Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Deep dive into design philosophy
- **[DESIGN_PATTERNS.md](DESIGN_PATTERNS.md)** - Best practices and patterns
- **[examples/integration/README.md](examples/integration/README.md)** - Complete integration guide
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - How to contribute

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./tests/integration/...

# Benchmark
go test -bench=. ./internal/engine/
```

---

## 🛣️ Roadmap

- [x] Core DAG execution engine
- [x] Memory, SQL, Redis storage
- [x] Built-in node library (8 types)
- [x] FunctionNode pattern
- [x] Signal/Human-in-loop
- [x] Event system
- [x] Prometheus metrics
- [x] Visual builder (Mermaid)
- [x] Saga/Compensation pattern
- [x] Complete working examples
- [ ] Web UI dashboard
- [ ] GraphQL API
- [ ] Workflow versioning
- [ ] Workflow templates marketplace
- [ ] Time-travel debugging

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 💬 Community & Support

- **GitHub Issues:** [Report bugs or request features](https://github.com/parevo/flow/issues)
- **Documentation:** [Full documentation](https://github.com/parevo/flow/wiki)
- **Examples:** [Working examples](examples/)

---

## 🙏 Acknowledgments

Inspired by:
- **Temporal** - For proving durable execution is essential
- **Airflow** - For popularizing DAG-based workflows
- **Cadence** - For the foundation of distributed orchestration
- **Asynq** - For showing Go can do simple, reliable job processing

Built with ❤️ by developers who believe **workflows should be simple, not complicated**.

---

<div align="center">

**⭐ Star us on GitHub — it motivates us to keep building!**

[Getting Started](#-quick-start) • [Examples](examples/) • [Documentation](ARCHITECTURE.md) • [Contributing](CONTRIBUTING.md)

</div>
