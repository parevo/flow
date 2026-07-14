# Parevo Flow

> Go uygulamaları için geliştirilmiş, DAG tabanlı, yüksek performanslı ve hafif bir iş akışı yönetim (workflow orchestration) motoru.

---

## 🎯 Giriş

**Parevo Flow**, Yönlü Döngüsüz Çizgeler (DAG) olarak modellenen karmaşık ve dağıtık iş süreçlerini yönetmek için tasarlanmıştır. İş akışı tasarımlarını basitleştirir, adım düzeyinde hata toleransı sunar, kalıcı durum yönetimi sağlar ve birden fazla işçi (worker) örneği üzerinde kolayca ölçeklenir.

### Temel Özellikler
*   **DAG Tabanlı Yönlendirme**: Grafik doğrulamaları (döngü kontrolleri) ve koşullu dallanmalar.
*   **Dağıtık Mimari**: Veritabanı veya önbellek üzerinden koordinasyon; harici bir mesaj kuyruğu sistemi gerektirmez.
*   **Hazır Görevler**: HTTP istekleri, Yapay Zeka (AI) LLM çağrıları, alt akışlar (sub-workflows), zamanlayıcılar, sinyaller, değişken atamaları ve özel Go fonksiyonları.
*   **Hata Toleransı**: Özelleştirilebilir yeniden deneme politikaları, üstel geri çekilme (exponential backoff) ve Saga örüntüsü (compensation) desteği.
*   **Esnek Güvenlik**: RBAC (Rol Tabanlı Yetkilendirme), metaveri takibi ve kiracı izolasyonu (tenant isolation) sağlayan pluggable yetkilendirme altyapısı.
*   **İzleme (Telemetry)**: Hazır Prometheus entegrasyonu.

---

## 🚀 Kurulum

Kütüphaneyi Go modüllerini kullanarak projenize dahil edebilirsiniz:

```bash
go get github.com/parevo/flow
```

---

## ⚙️ Gereksinimler

*   Go `1.23` veya daha yüksek bir sürüm.
*   *İsteğe bağlı* depolama katmanı: MySQL 5.7+, PostgreSQL 12+ veya Redis 6+ (geliştirme ve test süreçleri için in-memory depolama da mevcuttur).

---

## 💡 Hızlı Başlangıç

Bellek tabanlı çalışan basit bir iş akışı motoru başlatıp özel bir Go fonksiyonu tanımlayan ve çalıştıran örnek kod:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/parevo/flow"
)

func main() {
	// 1. Depolama ve Fonksiyon Kaydını Oluşturma
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	// 2. Özel bir Go fonksiyonu kaydetme
	registry.RegisterFunction("ProcessPayment", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		amount := input["amount"].(float64)
		fmt.Printf("💳 $%0.2f tutarındaki ödeme işleniyor...\n", amount)
		return map[string]interface{}{"status": "success", "tx_id": "tx_999"}, nil
	})

	// 3. Akışı Fluent Builder API ile Tanımlama
	wfBuilder := flow.NewWorkflow("payment-wf", "Payment Processor")
	wfBuilder.AddNode("pay", flow.NodeTypeFunction).
		WithConfig("function", "ProcessPayment").
		WithRetry(3)
	wf := wfBuilder.Build()

	ctx := context.Background()
	_ = engine.RegisterWorkflow(ctx, "default", wf)

	// 4. İşçi (Worker) Başlatma
	go engine.StartWorker(ctx, "default", "worker-1")

	// 5. Akışı Tetikleme
	input := []byte(`{"amount": 125.50}`)
	execID, _ := engine.Execute(ctx, "default", "payment-wf", input)

	// 6. Durumu Kontrol Etme
	time.Sleep(500 * time.Millisecond)
	exec, _ := engine.GetExecution(ctx, "default", execID)
	fmt.Printf("🏁 Akış durumu: %s | Çıktı: %s\n", exec.Status, exec.Output)
}
```
