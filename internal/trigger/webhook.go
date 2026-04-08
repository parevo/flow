package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/parevo/flow/internal/engine"
)

type WebhookManager struct {
	engine *engine.Engine
}

func NewWebhookManager(e *engine.Engine) *WebhookManager {
	return &WebhookManager{engine: e}
}

// ServeHTTP handles incoming webhooks: /webhooks/{namespace}/{workflow_id}
func (m *WebhookManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path. Expected /webhooks/{namespace}/{workflow_id}", http.StatusBadRequest)
		return
	}

	namespace := parts[1]
	workflowID := parts[2]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer func() { _ = r.Body.Close() }()

	execID, err := m.engine.StartWorkflow(context.Background(), namespace, workflowID, string(body))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"executionId": execID,
		"status":      "started",
	})
}
