# Parevo Flow

> A lightweight, high-performance, DAG-based workflow orchestration engine for Go applications.

---

## 🎯 Introduction

**Parevo Flow** is designed to execute complex, distributed business processes modeled as Directed Acyclic Graphs (DAGs). It simplifies workflow design, ensures task-level fault tolerance, provides state management, and scales gracefully across multiple worker instances.

### Key Features
*   **DAG-Based Routing**: Graph validations (cycle checks) and conditional routing.
*   **Distributed Architecture**: Coordination using a database or cache, no custom message broker required.
*   **Built-in Tasks**: Standard HTTP, LLM AI, nested sub-workflows, timers, signals, variable injection, and custom Go functions.
*   **Fault Tolerance**: Custom retry policies, exponential backoff, and Saga pattern compensations.
*   **Pluggable Security**: Pluggable authorization with RBAC, metadata tracking, and tenant isolation.
*   **Telemetry**: Prometheus metrics pre-integrated.

---

## 🚀 Installation

Install the library using Go modules:

```bash
go get github.com/parevo/flow
```

---

## ⚙️ Requirements

*   Go `1.23` or higher.
*   *Optional* storage backend: MySQL 5.7+, PostgreSQL 12+, or Redis 6+ (in-memory storage is also included for dev/testing).

---

## 💡 Quick Start

Here is a simple example starting a memory-backed workflow engine, registering a custom Go function, and executing it:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/parevo/flow"
)

func main() {
	// 1. Initialize Storage & Registry
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	// 2. Register a custom Go function handler
	registry.RegisterFunction("ProcessPayment", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		amount := input["amount"].(float64)
		fmt.Printf("💳 Processing payment of $%0.2f...\n", amount)
		return map[string]interface{}{"status": "success", "tx_id": "tx_999"}, nil
	})

	// 3. Define the Workflow using the Fluent Builder API
	wfBuilder := flow.NewWorkflow("payment-wf", "Payment Processor")
	wfBuilder.AddNode("pay", flow.NodeTypeFunction).
		WithConfig("function", "ProcessPayment").
		WithRetry(3)
	wf := wfBuilder.Build()

	ctx := context.Background()
	_ = engine.RegisterWorkflow(ctx, "default", wf)

	// 4. Start Worker Thread
	go engine.StartWorker(ctx, "default", "worker-1")

	// 5. Execute Workflow
	input := []byte(`{"amount": 125.50}`)
	execID, _ := engine.Execute(ctx, "default", "payment-wf", input)

	// 6. Monitor Status
	time.Sleep(500 * time.Millisecond)
	exec, _ := engine.GetExecution(ctx, "default", execID)
	fmt.Printf("🏁 Execution status: %s | Output: %s\n", exec.Status, exec.Output)
}
```
