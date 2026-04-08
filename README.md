# Parevo Flow 🌌🚀

**A Lightweight, High-Performance, and Intelligent Workflow Engine in Go.**

Parevo Flow is designed for modern SaaS architectures that require **extreme flexibility, enterprise-grade security, and intelligent decision-making** with **ZERO external dependencies** for observability.

---

## 🌟 Masterpiece Features

- **🧠 Intelligent Routing**: Built-in `ConditionNode` support for complex If-Else branching and decision trees.
- **🛡️ Enterprise Security**: Optional **AES-256-GCM** encryption for all sensitive workflow data (PII protection).
- **🏗️ Fluent Go Builder**: A type-safe, chainable DSL to build complex workflows directly in Go.
- **📈 Native Observability**: **Zero-dependency Prometheus metrics** exporter. No external libraries, just pure high-performance atomic counters.
- **⚡ Webhook & API Layer**: Trigger workflows via Webhooks and monitor them via a management REST API.
- **💎 Architectural Purity**: Isolated `internal` logic with a centralized `tests/` directory.
- **🚀 High-Load Ready**: Optimized with `SKIP LOCKED`, exponential backoff, and connection pool tuning.
- **🌍 Multi-Tenant Isolation**: Agnostic `Namespace` and `Labels` system for SaaS scalability.

---

## 🏗️ Fluent Builder Example

Build complex DAGs with zero effort:

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

## 📉 Zero-Dependency Monitoring (Prometheus)

Parevo Flow exposes native Prometheus metrics at `/metrics` using pure Go atomics. No external library overhead:
- `parevo_flow_tasks_processed_total`: Total throughput.
- `parevo_flow_tasks_failed_total`: Error rate tracking.
- `parevo_flow_active_workers`: Current scale of your worker fleet.

---

## 🛡️ Enterprise Security (AES-256)

Protect your data-at-rest with a single line of code:
```go
crypto, _ := storage.NewCrypto("64-char-hex-encryption-key...")
sqlStore.SetEncryption(crypto) // All Input/Output data will be AES-256-GCM secured!
```

---

## 📁 Directory Structure

```text
.
├── internal/
│   ├── builder/      # Fluent Go DSL
│   ├── engine/       # Core Orchestration (The Brain)
│   ├── storage/      # Persistence & AES-Crypto (The Vault)
│   ├── node/         # Logic Nodes (Wait, Condition, etc.)
│   └── trigger/      # API, Webhooks & Metrics (The Interface)
├── tests/            # Unified quality gate
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
