# API Başvurusu

Parevo Flow API'sinin temel bileşenleri ve yöntemleri hakkında detaylı kılavuz.

---

## ⚙️ Engine (Motor)

`Engine`, iş akışlarını kaydetmek, adımları yürütmek, sinyalleri işlemek ve akış durumlarını takip etmekle görevli ana bileşendir.

### Kurucu
```go
engine := flow.NewEngine(storage, registry)
```

### Metotlar

#### `RegisterWorkflow`
Belirli bir ad alanına (namespace) iş akışı şablonunu kaydeder.
```go
err := engine.RegisterWorkflow(ctx, "default", workflow)
```

#### `Execute`
Kayıtlı bir iş akışını tetikler. Girdi verilerini `[]byte` JSON formatında kabul eder.
```go
execID, err := engine.Execute(ctx, "default", "workflow-id", []byte(`{"param": "val"}`))
```

#### `GetExecution`
Kimliğe göre bir akışın çalışma durumunu ve çıkış detaylarını getirir.
```go
exec, err := engine.GetExecution(ctx, "default", execID)
```

#### `GetExecutionSteps`
Bir akışın altındaki tüm adımların durumunu, çıktılarını veya hata detaylarını sorgular.
```go
steps, err := engine.GetExecutionSteps(ctx, "default", execID)
```

#### `CancelExecution`
Çalışan bir akışı `CANCELLED` durumuna çekerek iptal eder.
```go
err := engine.CancelExecution(ctx, "default", execID)
```

#### `SignalExecution`
Bekleme durumundaki bir sinyal adımını (`flow.NodeTypeSignal`) dışarıdan veri göndererek devam ettirir.
```go
err := engine.SignalExecution(ctx, "default", execID, "node-id", `{"status":"approved"}`)
```

#### `StartWorker`
Belirtilen ad alanındaki adımları işleyen arka plan işçi (worker) döngüsünü başlatır.
```go
go engine.StartWorker(ctx, "default", "worker-id")
```

---

## 🗂️ Registry (Kayıt Defteri)

`Registry`, özel Go fonksiyonlarını ve özel geliştirilen `NodeExecutor` tiplerini saklar.

### Metotlar

#### `RegisterFunction`
Bir Go fonksiyonunu akış adımlarında kullanılmak üzere kaydeder.
```go
registry.RegisterFunction("SendNotification", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Özel mantık buraya yazılır
    return map[string]interface{}{"sent": true}, nil
})
```

#### `Register`
`NodeExecutor` arayüzünü uygulayan özel bir adım çalıştırıcısı kaydeder.
```go
registry.Register("custom_node_type", &MyCustomNodeExecutor{})
```

---

## 👷 Worker (İşçi)

`Worker`, hazır durumdaki adımları çekmek ve çalıştırmak için depolama katmanını periyodik sorgulayan arka plan sürecidir.

```go
worker := flow.NewWorker("worker-id", engine, registry, 100*time.Millisecond)
worker.SetNamespace("default")
worker.Start(ctx)
```

---

## 🎨 Workflow Builder (Akış Tasarımcısı)

`WorkflowBuilder`, akışları kod üzerinde akıcı bir arayüzle (fluent API) tasarlamayı sağlar.

### Metotlar

#### `NewWorkflow`
Yeni bir akış oluşturucu örneği başlatır.
```go
builder := flow.NewWorkflow("workflow-id", "Workflow Name")
```

#### `AddNode`
Akışa yeni bir adım ekler ve `NodeBuilder` döner.
```go
node := builder.AddNode("node-id", flow.NodeTypeFunction)
```

#### `WithConfig`
Adıma yapılandırma parametresi ekler.
```go
node.WithConfig("key", "value")
```

#### `WithRetry`
Adım başarısız olduğunda yapılacak maksimum deneme sayısını belirler.
```go
node.WithRetry(3)
```

#### `WithSaga`
Adım nihai olarak başarısız olduğunda çalıştırılacak telafi adımını tanımlar (Saga Örüntüsü).
```go
node.WithSaga("compensation-node-id")
```

#### `Then`
Geçerli adımı sıradaki hedef adıma bağlar.
```go
node.Then("next-node-id")
```

#### `If`
Geçerli adımı koşullu olarak hedef adıma bağlar (koşul doğruysa yönlenir).
```go
node.If("yes-node-id", "true")
```

#### `Connect`
İki adımı doğrudan birbirine bağlar.
```go
builder.Connect("source-id", "target-id")
```

#### `ConnectIf`
İki adımı koşullu olarak birbirine bağlar.
```go
builder.ConnectIf("source-id", "target-id", "condition")
```

#### `Build`
Tasarımı tamamlar ve derlenmiş `*flow.Workflow` nesnesini döner.
```go
wf := builder.Build()
```
