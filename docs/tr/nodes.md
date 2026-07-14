# Yerleşik Düğüm Tipleri

Parevo Flow, kutudan çıktığı haliyle birçok hazır adım düğümü tipiyle birlikte gelir.

---

## 🧩 Düğüm Detayları

### 1. `function`
Kayıt defterine (registry) eklenmiş özel Go fonksiyonlarını çalıştırır.
*   **Yapılandırma Seçenekleri**:
    *   `function` (string, zorunlu): Kayıtlı fonksiyonun adı.
*   **Örnek**:
    ```json
    {
      "id": "fetch_user",
      "type": "function",
      "config": {
        "function": "FetchUserFromDB"
      }
    }
    ```

### 2. `http`
Dış web servislerine HTTP istekleri gönderir.
*   **Yapılandırma Seçenekleri**:
    *   `url` (string, zorunlu): Hedef URL adresi.
    *   `method` (string, isteğe bağlı): HTTP metodu (`GET`, `POST` vb. Varsayılan: `GET`).
    *   `headers` (map, isteğe bağlı): İstek başlıkları (headers).
*   **Örnek**:
    ```json
    {
      "id": "fetch_status",
      "type": "http",
      "config": {
        "url": "https://api.example.com/v1/status",
        "method": "GET",
        "headers": {
          "Accept": "application/json"
        }
      }
    }
    ```

### 3. `condition`
Karşılaştırma mantığı işletir ve akışı sonuca göre `true` veya `false` dallarına yönlendirir.
*   **Yapılandırma Seçenekleri**:
    *   `variable` (string, zorunlu): Girdide okunacak alan.
    *   `operator` (string, zorunlu): `==`, `!=`, `>`, `<`, `>=`, `<=`, `contains`, `not_contains`.
    *   `value` (any, zorunlu): Karşılaştırılacak değer.
*   **Örnek**:
    ```json
    {
      "id": "check_age",
      "type": "condition",
      "config": {
        "variable": "age",
        "operator": ">=",
        "value": 18
      }
    }
    ```

### 4. `switch`
Girdi alanının değerine göre çok yönlü dallanma (multi-way branching) sağlar.
*   **Yapılandırma Seçenekleri**:
    *   `variable` (string, zorunlu): Girdide okunacak alan.
    *   `cases` (map, zorunlu): Değer-dal eşlemeleri.
    *   `default` (string, isteğe bağlı): Hiçbir eşleşme olmadığında seçilecek varsayılan dal (Varsayılan: `default`).
*   **Örnek**:
    ```json
    {
      "id": "route_role",
      "type": "switch",
      "config": {
        "variable": "role",
        "cases": {
          "admin": "admin-branch",
          "user": "user-branch"
        },
        "default": "guest-branch"
      }
    }
    ```

### 5. `signal`
Akışı duraklatır ve devam etmek için REST API üzerinden dışarıdan bir sinyal gelmesini bekler.
*   **Yapılandırma Seçenekleri**:
    *   `timeout` (string, isteğe bağlı): Zaman aşımı süresi (örn. `24h`, `7d`). Aşılırsa adım hata verir.
*   **Örnek**:
    ```json
    {
      "id": "wait_approval",
      "type": "signal",
      "config": {
        "timeout": "48h"
      }
    }
    ```

### 6. `subworkflow`
Başka bir iş akışını alt akış (child workflow) olarak tetikler ve tamamlanmasını bekler.
*   **Yapılandırma Seçenekleri**:
    *   `workflowId` (string, zorunlu): Tetiklenecek alt akışın ID'si.
    *   `namespace` (string, isteğe bağlı): Alt akışın çalışacağı ad alanı.
*   **Örnek**:
    ```json
    {
      "id": "run_onboarding",
      "type": "subworkflow",
      "config": {
        "workflowId": "employee-onboarding",
        "namespace": "default"
      }
    }
    ```

### 7. `ai`
OpenAI, Anthropic veya Google Gemini modellerine prompt istekleri gönderir.
*   **Yapılandırma Seçenekleri**:
    *   `provider` (string, zorunlu): `openai`, `anthropic` veya `gemini`.
    *   `api_key` (string, zorunlu): API anahtarı.
    *   `model` (string, zorunlu): Model adı (örn. `gpt-4o`, `claude-3-5-sonnet`, `gemini-1.5-pro`).
    *   `prompt` (string, zorunlu): `{{.alan}}` formatında değişkenleri destekleyen prompt şablonu.
    *   `system_prompt` (string, isteğe bağlı): Sistem yönergesi.
    *   `temperature` (float, isteğe bağlı): Yaratıcılık katsayısı (0.0-2.0. Varsayılan: `0.7`).
    *   `max_tokens` (int, isteğe bağlı): Maksimum yanıt token limiti (Varsayılan: `1000`).
    *   `result_key` (string, isteğe bağlı): Yanıtın yazılacağı çıktı alanı anahtarı (Varsayılan: `ai_response`).
*   **Örnek**:
    ```json
    {
      "id": "ai_summarize",
      "type": "ai",
      "config": {
        "provider": "openai",
        "api_key": "YOUR_OPENAI_KEY",
        "model": "gpt-4o",
        "prompt": "Bu müşteri bildirimini özetle: {{.feedback}}",
        "result_key": "summary"
      }
    }
    ```

### 8. `notify`
Dış URL adreslerine Go şablonları (Go templates) ile oluşturulmuş HTTP webhook bildirimleri gönderir.
*   **Yapılandırma Seçenekleri**:
    *   `url` (string, zorunlu): Hedef URL adresi (`{{.alan}}` içerebilir).
    *   `method` (string, isteğe bağlı): HTTP metodu (Varsayılan: `POST`).
    *   `body` (string, isteğe bağlı): Webhook gövdesi şablonu. Boşsa doğrudan girdiyi iletir.
*   **Örnek**:
    ```json
    {
      "id": "slack_notify",
      "type": "notify",
      "config": {
        "url": "https://hooks.slack.com/services/T00/B00/X00",
        "method": "POST",
        "body": "{\"text\": \"Yeni kayıt: {{.user.name}} ({{.user.email}})\"}"
      }
    }
    ```

### 9. `transform`
JSON veri context'ini Go şablonları yardımıyla dönüştürür ve yeniden şekillendirir.
*   **Yapılandırma Seçenekleri**:
    *   `mapping` (map, zorunlu): Anahtar-şablon eşleştirmeleri.
*   **Örnek**:
    ```json
    {
      "id": "format_payload",
      "type": "transform",
      "config": {
        "mapping": {
          "fullName": "{{.firstName}} {{.lastName}}",
          "contactEmail": "{{.email}}"
        }
      }
    }
    ```

### 10. `wait`
Akışın belirli bir süre duraklamasını (delay) sağlar.
*   **Yapılandırma Seçenekleri**:
    *   `duration` (string, zorunlu): Süre ifadesi (örn. `5s`, `10m`, `2h`).
*   **Örnek**:
    ```json
    {
      "id": "wait_5_sec",
      "type": "wait",
      "config": {
        "duration": "5s"
      }
    }
    ```

### 11. `setvariable`
JSON veri bağlamına (payload) değişken ekler veya mevcut olanları günceller.
*   **Yapılandırma Seçenekleri**:
    *   `variables` (map, zorunlu): Eklenecek değişkenler.
*   **Örnek**:
    ```json
    {
      "id": "set_meta",
      "type": "setvariable",
      "config": {
        "variables": {
          "status": "pending_billing",
          "processed_by": "engine-v1"
        }
      }
    }
    ```

### 12. `log`
Konsola log mesajları yazdırır.
*   **Yapılandırma Seçenekleri**:
    *   `message` (string, zorunlu): Yazdırılacak mesaj şablonu.
*   **Örnek**:
    ```json
    {
      "id": "audit_log",
      "type": "log",
      "config": {
        "message": "İlgili adıma başarıyla ulaşıldı!"
      }
    }
    ```
