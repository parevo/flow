package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/node"
	"github.com/parevo/flow/internal/storage"
	"github.com/parevo/flow/internal/storage/memory"
	"github.com/parevo/flow/internal/trigger"
)

// PassThroughNode just returns the input as is
type PassThroughNode struct{}
func (n *PassThroughNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	return input, nil
}
func (n *PassThroughNode) Validate(config map[string]interface{}) error {
	return nil
}

func TestIntelligenceAndSecurity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Setup Storage
	memStore := memory.NewMemoryStorage()
	reg := engine.NewRegistry()
	reg.Register("pass", &PassThroughNode{})
	reg.Register("log", node.NewLogNode())
	reg.Register("condition", &node.ConditionNode{})
	
	eng := engine.NewEngine(memStore, reg)
	worker := engine.NewWorker("master-worker", eng, reg, 100*time.Millisecond)

	namespace := "enterprise-org"

	// 2. Define Intelligent Workflow: Pass -> Condition -> (True: Log A / False: Log B)
	wf := &models.Workflow{
		ID: "intel-wf",
		Nodes: []models.Node{
			{ID: "start", Type: "pass", Config: map[string]interface{}{}},
			{ID: "decide", Type: "condition", Config: map[string]interface{}{
				"variable": "amount", "operator": "==", "value": 100,
			}},
			{ID: "path-true", Type: "log", Config: map[string]interface{}{"message": "Path TRUE"}},
			{ID: "path-false", Type: "log", Config: map[string]interface{}{"message": "Path FALSE"}},
		},
		Edges: []models.Edge{
			{ID: "e1", SourceID: "start", TargetID: "decide"},
			{ID: "e-true", SourceID: "decide", TargetID: "path-true", Condition: "true"},
			{ID: "e-false", SourceID: "decide", TargetID: "path-false", Condition: "false"},
		},
	}
	memStore.SaveWorkflow(ctx, namespace, wf)

	// 3. Test Webhook Trigger
	webhookMgr := trigger.NewWebhookManager(eng)
	reqBody := `{"amount": 100, "user": "ahmet"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/enterprise-org/intel-wf", strings.NewReader(reqBody))
	rr := httptest.NewRecorder()

	webhookMgr.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Webhook failed: %s", rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	execID := resp["executionId"]

	// 4. Run Worker
	go worker.Start(ctx)

	// 5. Verify Path TRUE was taken
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		exec, _ := memStore.GetExecution(ctx, namespace, execID)
		if exec != nil && exec.Status == models.TaskCompleted {
			steps, _ := memStore.GetExecutionSteps(ctx, namespace, execID)
			
			foundTrue := false
			for _, s := range steps {
				if s.NodeID == "path-true" { foundTrue = true }
			}

			if !foundTrue {
				t.Fatalf("Routing failed! PathTrue was not executed")
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("Workflow did not complete")
}

func TestEncryption(t *testing.T) {
	keyHex := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	c, err := storage.NewCrypto(keyHex)
	if err != nil {
		t.Fatalf("Failed to create crypto: %v", err)
	}

	plaintext := "sensitive user data"
	ciphertext, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if ciphertext == plaintext {
		t.Fatal("Ciphertext is same as plaintext")
	}

	decrypted, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("Decrytped data mismatch: got %s, want %s", decrypted, plaintext)
	}
}
