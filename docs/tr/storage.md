# Veri Depolama & Güvenlik

Parevo Flow; akış durumlarının kalıcılığını, kilit mekanizmalarını ve metaverileri soyut bir `Storage` arayüzü ile yönetir.

---

## 💾 Depolama Seçenekleri

### 1. SQL Depolama (MySQL & PostgreSQL)
Üretim (production) ortamları için önerilir. SQL depoları, ilk bağlantıda şema göçlerini (DDL migrations) otomatik olarak yürütür ve işçi (worker) senkronizasyonunu yönetmek için veritabanı kilitlerini kullanır.

*   **Eşzamanlılık Kilitleme**: Görevleri çekerken veritabanı düzeyinde satır kilitleme (`SELECT ... FOR UPDATE SKIP LOCKED`) yöntemi kullanılır. Bu sayede, farklı sunucularda çalışan işçi süreçlerinin aynı adımı aynı anda devralması kesinlikle engellenir.
*   **MySQL Kullanımı**:
    ```go
    import (
    	_ "github.com/go-sql-driver/mysql"
    	"github.com/jmoiron/sqlx"
    	"github.com/parevo/flow"
    )

    db, _ := sqlx.Connect("mysql", "user:pass@tcp(localhost:3306)/db?parseTime=true")
    storage, _ := flow.NewMySQLStorage(db)
    ```
*   **PostgreSQL Kullanımı**:
    ```go
    import (
    	_ "github.com/lib/pq"
    	"github.com/jmoiron/sqlx"
    	"github.com/parevo/flow"
    )

    db, _ := sqlx.Connect("postgres", "postgres://user:pass@localhost/db?sslmode=disable")
    storage, _ := flow.NewPostgreSQLStorage(db)
    ```

### 2. Redis Depolama
Düşük gecikme süreli akış sistemleri için uygun, yüksek performanslı anahtar-değer depolama seçeneğidir.
*   **Eşzamanlılık Kilitleme**: Görevleri planlamak için bir Redis Sorted Set (ZSET) kullanır. Adımı kuyruktan çekme ve `RUNNING` durumuna çekme aşamaları, Redis üzerinde koşan tek bir atomik **Lua Betiği** olarak uygulanmıştır. Bu sayede thread-safe ve izole bir işlem yapısı sağlanır.
*   **Kullanım**:
    ```go
    storage := flow.NewRedisStorage("localhost:6379", "password", 0)
    ```

### 3. Memory Depolama
`sync.RWMutex` ile korunan bellek içi (in-memory) depolama katmanıdır. Yerel testler, prototipler veya birim test doğrulamaları için mükemmeldir.
*   **Kullanım**:
    ```go
    storage := flow.NewMemoryStorage()
    ```

---

## 🔒 Veri Şifreleme (AES-GCM)

Güvenlik uyumluluk standartlarını (örn. PCI-DSS, KVKK, GDPR) karşılamak için Parevo Flow, akış verilerinin şifrelenerek saklanmasını destekler. Bu özellik aktif edildiğinde, akış girdileri, adım çıktıları ve hata detayları veritabanına yazılmadan önce şifrelenir ve okunurken otomatik olarak deşifre edilir.

### Kullanım
```go
// 1. 32-byte boyutunda anahtarla Crypto yardımcısını oluşturun
crypto, err := flow.NewCrypto("your-32-byte-encryption-key-here")
if err != nil {
    log.Fatal("Geçersiz şifreleme anahtarı")
}

// 2. SQL Depolama katmanına şifreleyiciyi tanımlayın
sqlStorage, ok := storage.(*sql.SQLStorage)
if ok {
    sqlStorage.SetEncryption(crypto)
}
```
*Şifreleme özelliği, `redisStorage.SetEncryption(crypto)` çağrısı ile Redis depolama katmanında da kullanılabilmektedir.*
