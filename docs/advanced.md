# Advanced Features

Learn about advanced enterprise features: Pluggable Authorization, Cron Tickers, Webhooks, and Monitoring.

---

## 🔒 Pluggable Authorization

Parevo Flow enforces access controls on workflow definitions using a pluggable `AuthProvider` interface.

```go
type AuthProvider interface {
    CheckAccess(ctx context.Context, resource string, action string) error
}
```

### 1. No Auth (Development)
By default, the engine is created without an auth provider, allowing all actions:
```go
engine := flow.NewEngine(storage, registry)
```

### 2. Custom RBAC & Tenant Isolation
You can pass credentials (like JWT tokens, User ID, Team ID) using `context.Context`, and define custom checks inside `CheckAccess`:

```go
type CustomAuth struct{}

func (a *CustomAuth) CheckAccess(ctx context.Context, resource string, action string) error {
	// 1. Retrieve tenant info from context
	customerID, _ := ctx.Value("customer_id").(string)
	if customerID == "" {
		return errors.New("unauthorized: missing customer_id")
	}

	// 2. Format: "workflow:namespace:workflow-id"
	// Parse resource details to verify ownership
	// ...
	
	return nil
}

// Register Auth Provider
engine.SetAuthProvider(&CustomAuth{})
```

---

## ⏱️ Scheduling (Cron Manager)

Run workflows periodically using Cron expressions. The `CronManager` wraps the engine and schedules tasks automatically.

```go
import (
	"log/slog"
	"os"
	"github.com/parevo/flow"
)

// 1. Create Cron Manager
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
cronMgr := flow.NewCronManager(engine, logger)

// 2. Add schedules: namespace, workflowID, cronExpression, input string
_, err := cronMgr.AddSchedule("default", "db-backup-wf", "0 2 * * *", `{"type":"daily"}`)

// 3. Start cron scheduler loop
cronMgr.Start()
defer cronMgr.Stop()
```

---

## 📡 Webhook Manager

Launch workflows dynamically using standard HTTP POST requests. The `WebhookManager` handles routing and triggers executions immediately.

```go
webhookMgr := flow.NewWebhookManager(engine)

// Bind endpoint mapping: /webhooks/{namespace}/{workflow_id}
http.Handle("/webhooks/", webhookMgr)
log.Fatal(http.ListenAndServe(":8080", nil))
```
Trigger execution via terminal:
```bash
curl -X POST http://localhost:8080/webhooks/default/my-workflow -d '{"data": "value"}'
```

---

## 📊 Telemetry & Monitoring

### 1. Prometheus Metrics
Metrics are pre-registered and served using standard handlers.

```go
import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

http.Handle("/metrics", promhttp.Handler())
```

*   **Workflows Metrics**:
    *   `flow_workflows_started_total`
    *   `flow_workflows_completed_total`
    *   `flow_workflows_failed_total`
*   **Step Metrics**:
    *   `flow_steps_processed_total` (Labels: `namespace`, `node_type`, `status`)
    *   `flow_step_duration_seconds` (Labels: `namespace`, `node_type`)
*   **Worker Metrics**:
    *   `flow_active_workers` (Labels: `worker_id`)

### 2. Structured Logging
Inject custom `slog.Logger` handles into the engine:
```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
engine.WithLogger(logger)
```
