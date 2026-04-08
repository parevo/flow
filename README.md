# Parevo Flow 🌌🚀

**A Lightweight, High-Performance, and Intelligent Workflow Engine in Go.**

Parevo Flow is an enterprise-grade DAG orchestration engine designed for modern SaaS architectures. It combines **extreme flexibility, bulletproof reliability, and intelligent decision-making** with zero external dependencies for core observability.

---

## 🌟 Gold Standard Features

- **🧠 Intelligent Routing**: Built-in `ConditionNode` support for complex If-Else branching and dynamic decision trees.
- **🛡️ Enterprise Security**: Optional **AES-256-GCM** encryption for all sensitive PII data-at-rest.
- **🧟‍♂️ Self-Healing (Zombie Recovery)**: Automatic recovery from worker crashes. Stalled tasks are reclaimed after a 5-minute visibility timeout.
- **🛑 Execution Cancellation**: Instantly stop unwanted or malfunctioning workflows via the management API.
- **🏗️ Fluent Go Builder**: A type-safe, chainable DSL to build complex workflows directly in Go.
- **📈 Native Observability**: **Zero-dependency Prometheus metrics** exporter for real-time throughput and health tracking.
- **🚀 High-Load Performance**: Optimized SQL with **Composite Indexes** and `SKIP LOCKED` concurrency.
- **🌍 Multi-Tenant Isolation**: Native `Namespace` and `Labels` support for ultimate scalability.

---

## 🏗️ Fluent Builder Example

Define complex business logic with zero friction:

```go
wf := builder.NewWorkflow("signup-flow", "User Signup")
    .AddNode("validate", "http").WithConfig("url", "https://api.check.com")
    .Then("check-status")
    .AddNode("check-status", "condition").WithConfig("variable", "valid", "operator", "==", "value", true)
    .If("welcome-email", "true")
    .If("flag-user", "false")
    .Build()
```

---

## 🧟‍♂️ Reliability & Self-Healing

Parevo Flow is built for production stability. If a worker node crashes mid-task:
1. The engine detects the **Zombie Task** using the `updated_at` heartbeat.
2. The task is released back to the pool after the **Visibility Timeout**.
3. Another worker automatically claims and continues the execution.

---

## 🛡️ Enterprise Security

Securing customer data is a single command away:
```go
crypto, _ := storage.NewCrypto("64-char-hex-encryption-key...")
sqlStore.SetEncryption(crypto) // All Input/Output data is now AES-256 secured!
```

---

## 📈 Monitoring (Prometheus)

Expose industry-standard metrics at `/metrics` with **zero external libraries**:
- `parevo_flow_tasks_processed_total`: Total throughput.
- `parevo_flow_tasks_failed_total`: Error rate tracking.
- `parevo_flow_active_workers`: Current scale of your worker fleet.

---

## 📁 Directory Structure

```text
.
├── internal/
│   ├── builder/      # Fluent Go DSL Builders
│   ├── engine/       # Core Brain & DAG Orchestration
│   ├── storage/      # The Vault (SQL Drivers & AES-Crypto)
│   ├── node/         # Logic Nodes (Wait, Condition, HTTP, etc.)
│   └── trigger/      # Interface (API, Webhooks & Metrics)
├── tests/            # Professional Quality Gate
└── README.md         # This masterpiece
```

---

## 🚀 Quick Start
```bash
go get github.com/parevo/flow
```

---

## 📄 License
Distributed under the **MIT License**.

---
**Parevo Flow** - *Built with ❤️ for the Gopher community by Ahmet Can Bilgay.*
