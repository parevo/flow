# Parevo Flow 🌌🚀

**A Lightweight, High-Performance, and Intelligent Workflow Engine in Go.**

Parevo Flow is designed for modern SaaS architectures that require **extreme flexibility, enterprise-grade security, and intelligent decision-making** without the overhead of heavy infrastructure.

---

## 🌟 Masterpiece Features

- **🧠 Intelligent Routing**: Built-in `ConditionNode` support for complex If-Else branching and decision trees.
- **🛡️ Enterprise Security**: Optional **AES-256-GCM** encryption for all sensitive workflow data (PII protection).
- **🏗️ Fluent Go Builder**: A type-safe, chainable DSL to build complex workflows directly in Go.
- **📈 Real-time Observability**: Native **Prometheus** metrics integration for monitoring throughput, failures, and latency.
- **⚡ Webhook & API Layer**: Trigger workflows via Webhooks and monitor them via a management REST API.
- **💎 Architectural Purity**: Isolated `internal` logic with a centralized `tests/` directory.
- **🚀 High-Load Ready**: Optimized with `SKIP LOCKED`, exponential backoff, and connection pool tuning.
- **🌍 Multi-Tenant Isolation**: Agnostic `Namespace` and `Labels` system for SaaS scalability.

---

## 🏗️ Fluent Builder Example

Build complex DAGs without touching raw JSON:

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

## 📈 Monitoring (Prometheus)

Parevo Flow exposes industry-standard metrics at `/metrics`:
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
│   ├── builder/      # Fluent Go DSL (The Painter)
│   ├── engine/       # Core Orchestration (The Brain)
│   ├── storage/      # Persistence & AES-Crypto (The Vault)
│   ├── node/         # Logic Nodes (Wait, Condition, HTTP, etc.)
│   └── trigger/      # API, Webhooks & Metrics (The Interface)
├── tests/            # Unified quality gate (Smoke & Integration)
└── README.md         # This masterpiece
```

---

## 🚀 Quick Start
```bash
go get github.com/parevo/flow
```

---

## 🤝 Contributing
Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## 📄 License
Distributed under the **MIT License**. See `LICENSE` for more information.

---
**Parevo Flow** - *Built with ❤️ for the Gopher community by Ahmet Can Bilgay.*
