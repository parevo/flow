package trigger

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/parevo/flow/internal/engine"
)

// APIManager handles management API requests
type APIManager struct {
	webhookMgr *WebhookManager
}

func NewAPIManager(wm *WebhookManager) *APIManager {
	return &APIManager{webhookMgr: wm}
}

func (m *APIManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	
	// 1. Prometheus Metrics (Zero-Dependency Implementation)
	if path == "metrics" {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Write([]byte(engine.GetTelemetry().ToPrometheusFormat()))
		return
	}

	// 2. Health Check
	if path == "health" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	// 3. Management APIs: /api/{namespace}/executions...
	if strings.HasPrefix(path, "api/") {
		m.handleManagementAPI(w, r, path)
		return
	}

	// 4. Webhook Trigger logic (existing)
	if strings.HasPrefix(path, "webhooks/") {
		m.webhookMgr.ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}

func (m *APIManager) handleManagementAPI(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid API path", http.StatusBadRequest)
		return
	}

	namespace := parts[1]
	resource := parts[2]

	switch resource {
	case "executions":
		if len(parts) == 3 {
			// List executions (Placeholder: actually needs storage call)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"namespace": namespace, "executions": []string{}})
			return
		}
		if len(parts) == 4 {
			// Get Execution status
			execID := parts[3]
			exec, err := m.webhookMgr.engine.GetExecutionStatus(r.Context(), namespace, execID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(exec)
			return
		}
	}

	http.NotFound(w, r)
}
