# Storage Backends & Security

Parevo Flow isolates state persistence, locking mechanics, and metadata storage into a pluggable `Storage` interface.

---

## 💾 Storage Implementations

### 1. SQL Storage (MySQL & PostgreSQL)
Designed for production setups. SQL backends automatically execute DDL migrations on initialization and use standard transaction isolation to handle multi-worker synchronization.

*   **Concurrency Locking**: Uses database-level locks (`SELECT ... FOR UPDATE SKIP LOCKED`) during the task claim phase. This guarantees that multiple worker processes running across different machines will never process the same step concurrently.
*   **MySQL Usage**:
    ```go
    import (
    	_ "github.com/go-sql-driver/mysql"
    	"github.com/jmoiron/sqlx"
    	"github.com/parevo/flow"
    )

    db, _ := sqlx.Connect("mysql", "user:pass@tcp(localhost:3306)/db?parseTime=true")
    storage, _ := flow.NewMySQLStorage(db)
    ```
*   **PostgreSQL Usage**:
    ```go
    import (
    	_ "github.com/lib/pq"
    	"github.com/jmoiron/sqlx"
    	"github.com/parevo/flow"
    )

    db, _ := sqlx.Connect("postgres", "postgres://user:pass@localhost/db?sslmode=disable")
    storage, _ := flow.NewPostgreSQLStorage(db)
    ```

### 2. Redis Storage
High-performance key-value backend suited for low-latency workflow systems.
*   **Concurrency Locking**: Utilizes a Redis Sorted Set (ZSET) to schedule steps. Task claiming is implemented as an atomic **Lua Script** running inside Redis, ensuring that popping a step and transitioning it to `RUNNING` is thread-safe and isolated.
*   **Usage**:
    ```go
    storage := flow.NewRedisStorage("localhost:6379", "password", 0)
    ```

### 3. Memory Storage
RWMutex-backed in-memory store. Recommended for local testing, prototyping, and mock execution verification.
*   **Usage**:
    ```go
    storage := flow.NewMemoryStorage()
    ```

---

## 🔒 Payload Encryption (AES-GCM)

To meet security compliance standards (e.g., PCI-DSS, GDPR), Parevo Flow supports transparent data-at-rest encryption of execution payloads. When enabled, execution inputs, step outputs, and error details are encrypted before being written to disk/database, and decrypted automatically on read.

### Usage
```go
// 1. Instantiate the Crypto helper with a 32-byte key
crypto, err := flow.NewCrypto("your-32-byte-encryption-key-here")
if err != nil {
    log.Fatal("Invalid encryption key")
}

// 2. Cast and attach the Crypto handler to your SQL storage backend
sqlStorage, ok := storage.(*sql.SQLStorage)
if ok {
    sqlStorage.SetEncryption(crypto)
}
```
*Encryption is also supported on the Redis backend via `redisStorage.SetEncryption(crypto)`.*
