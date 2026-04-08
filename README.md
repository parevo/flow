# Parevo Flow 🌌🚀

**A Lightweight, High-Performance, and Intelligent Workflow Engine in Go.**

Parevo Flow is designed for modern SaaS architectures that require **extreme flexibility, enterprise-grade security, and intelligent decision-making** without the overhead of heavy infrastructure.

---

## 🌟 Masterpiece Features

- **🧠 Intelligent Routing**: Built-in `ConditionNode` support for complex If-Else branching and decision trees.
- **🛡️ Enterprise Security**: Optional **AES-256-GCM** encryption for all sensitive workflow data (PII protection).
- **⚡ Webhook Automation**: Trigger any workflow via HTTP POST requests using the integrated Webhook Layer.
- **💎 Architectural Purity**: Isolated `internal` logic with a centralized `tests/` directory for maximum maintainability.
- **🚀 High-Load Ready**: Optimized with `SKIP LOCKED` polling, exponential backoff retries, and database connection pool tuning.
- **🌍 Multi-Tenant Isolation**: Agnostic `Namespace` and `Labels` system for ultimate SaaS scalability.
- **🔌 Multi-DB Support**: Native support for **MySQL** and **PostgreSQL**.

---

## 📁 Directory Structure

```text
.
├── internal/
│   ├── engine/       # Core DAG and Orchestration Beyni
│   ├── storage/      # Persistence, SQL Drivers & AES-Crypto
│   ├── node/         # Pre-built Nodes (Log, HTTP, Wait, Condition)
│   └── trigger/      # Automation Triggers (Webhook Layer)
├── tests/            # Unified quality gate (Integration & Smoke Tests)
├── LICENSE           # MIT
└── README.md         # This masterpiece
```

---

## 🚀 Quick Start

### 1. Installation
```bash
go get github.com/parevo/flow
```

### 2. Basic Workflow with Decision Logic
```go
wf := &models.Workflow{
    ID: "onboarding-wf",
    Nodes: []models.Node{
        {ID: "check", Type: "condition", Config: map[string]interface{}{
            "variable": "is_vip", "operator": "==", "value": true,
        }},
        {ID: "vip-welcome", Type: "log", Config: map[string]interface{}{"message": "Welcome VIP!"}},
        {ID: "std-welcome", Type: "log", Config: map[string]interface{}{"message": "Welcome standard user."}},
    },
    Edges: []models.Edge{
        {SourceID: "check", TargetID: "vip-welcome", Condition: "true"},
        {SourceID: "check", TargetID: "std-welcome", Condition: "false"},
    },
}
```

### 3. Enabling Enterprise Security (AES-256)
Protect your data-at-rest with a single line of code:
```go
crypto, _ := storage.NewCrypto("64-char-hex-encryption-key...")
storage.SetEncryption(crypto) // All Input/Output data will be AES-256-GCM secured!
```

---

## 🏗️ Production Readiness (High-Load)

For handling millions of workloads, ensure your DB connection pool is tuned:

```go
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

## 🤝 Contributing
Contributions are welcome! Parevo Flow is community-driven and open to new nodes and triggers.

## 📄 License
Distributed under the **MIT License**. See `LICENSE` for more information.

---
**Parevo Flow** - *Built with ❤️ for the Gopher community by Ahmet Can Bilgay.*
