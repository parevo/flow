package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/node"
	"github.com/parevo/flow/internal/storage/memory"
)

// LogNode is a simple action node that prints to the console
type LogNode struct{}
func (n *LogNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	msg := config["message"]
	if msg == nil {
		msg = "No message provided"
	}
	fmt.Printf("\n📝 [LOG NODE] executing. Message: %v\n", msg)
	if input == "" { input = "{}" }
	var m map[string]interface{}
	json.Unmarshal([]byte(input), &m)
	delete(m, "branch") // Strip internal routing tag
	out, _ := json.Marshal(m)
	return string(out), nil
}
func (n *LogNode) Validate(config map[string]interface{}) error { return nil }

type ProgressNode struct{}
func (n *ProgressNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	stepName, _ := config["step_name"].(string)
	progress, _ := config["progress"].(string)
	fmt.Printf("\n📊 [Progress] Adım: %s -> Oran: %s\n", stepName, progress)
	if input == "" { input = "{}" }
	var m map[string]interface{}
	json.Unmarshal([]byte(input), &m)
	delete(m, "branch") // Strip internal routing tag
	out, _ := json.Marshal(m)
	return string(out), nil
}
func (n *ProgressNode) Validate(config map[string]interface{}) error { return nil }

type UINode struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Data map[string]interface{} `json:"data"`
}

type UIEdge struct {
    Source      string `json:"source"`
    Target      string `json:"target"`
    Condition   string `json:"condition"`
}

type UIWorkflowPayload struct {
    Nodes []UINode `json:"nodes"`
    Edges []UIEdge `json:"edges"`
}

type SignalPayload struct {
    ExecutionID string `json:"execution_id"`
    NodeID      string `json:"node_id"`
    Data        string `json:"data"`
}

func main() {
    // 1. Setup Engine
    memStore := memory.NewMemoryStorage()
    registry := engine.NewRegistry()
    registry.Register("log", &LogNode{})
    registry.Register("progress", &ProgressNode{})
    registry.Register("wait", &node.WaitNode{})
    registry.Register("http", node.NewHTTPNode())
    registry.Register("condition", &node.ConditionNode{})
    registry.Register("signal", &node.SignalNode{})
    
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
    eng := engine.NewEngine(memStore, registry).WithLogger(logger)
    
    ctx := context.Background()

    // 2. Start Worker for background execution
    w := engine.NewWorker("visual-worker-1", eng, registry, 100*time.Millisecond)
    go w.Start(ctx)

    // 3. HTTP Endpoints
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "index.html")
    })

    http.HandleFunc("/api/run", func(wr http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(wr, "Only POST allowed", http.StatusMethodNotAllowed)
            return
        }
        
        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(wr, "Error reading body", http.StatusBadRequest)
            return
        }

        var payload UIWorkflowPayload
        err = json.Unmarshal(body, &payload)
        if err != nil {
            http.Error(wr, "Invalid JSON", http.StatusBadRequest)
            return
        }

        wf := &models.Workflow{
            ID:          "visual-wf-1",
            Name:        "Visual Drag Drop Workflow",
            Description: "Generated from browser",
        }

        // Convert UI Nodes to Parevo Nodes
        for _, un := range payload.Nodes {
            parevoNode := models.Node{
                ID:   un.ID,
                Type: un.Name,
                Config: un.Data,
            }
            wf.Nodes = append(wf.Nodes, parevoNode)
        }

        // Convert UI Edges to Parevo Edges
        for _, ue := range payload.Edges {
            wf.Edges = append(wf.Edges, models.Edge{
                SourceID:  ue.Source,
                TargetID:  ue.Target,
                Condition: ue.Condition,
            })
        }

        memStore.SaveWorkflow(ctx, "visual-ns", wf)
        
        // Execute the dynamically built workflow!
        execID, err := eng.StartWorkflow(ctx, "visual-ns", "visual-wf-1", "user-ui")
        if err != nil {
            http.Error(wr, err.Error(), http.StatusInternalServerError)
            return
        }

        fmt.Printf("\n🚀 UI Workflow submitted successfully! (ExecID: %s)\n", execID)
        wr.Header().Set("Content-Type", "application/json")
        wr.Write([]byte(`{"status":"success", "execution_id":"` + execID + `"}`))
    })

    http.HandleFunc("/api/signal", func(wr http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(wr, "Only POST allowed", http.StatusMethodNotAllowed)
            return
        }
        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(wr, "Error reading body", http.StatusBadRequest)
            return
        }
        var sp SignalPayload
        if err := json.Unmarshal(body, &sp); err != nil {
            http.Error(wr, "Invalid JSON", http.StatusBadRequest)
            return
        }
        
        err = eng.SignalExecution(ctx, "visual-ns", sp.ExecutionID, sp.NodeID, sp.Data)
        if err != nil {
            http.Error(wr, err.Error(), http.StatusInternalServerError)
            return
        }

        fmt.Printf("\n⚡ SIGNAL RECEIVED! Unblocking execution %s at node %s...\n", sp.ExecutionID, sp.NodeID)
        wr.Header().Set("Content-Type", "application/json")
        wr.Write([]byte(`{"status":"success"}`))
    })

    http.HandleFunc("/api/status", func(wr http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
            http.Error(wr, "Only GET allowed", http.StatusMethodNotAllowed)
            return
        }
        execID := r.URL.Query().Get("execution_id")
        if execID == "" {
            http.Error(wr, "execution_id required", http.StatusBadRequest)
            return
        }

        steps, err := memStore.GetExecutionSteps(ctx, "visual-ns", execID)
        if err != nil {
            http.Error(wr, err.Error(), http.StatusInternalServerError)
            return
        }

        result := make(map[string]map[string]string)
        for _, step := range steps {
            result[step.NodeID] = map[string]string{
                "status": string(step.Status),
                "error":  step.Error,
            }
        }

        respBytes, _ := json.Marshal(result)
        wr.Header().Set("Content-Type", "application/json")
        wr.Write(respBytes)
    })

    fmt.Println("🌟 Parevo Flow Visual Studio is running!")
    fmt.Println("👉 Open your browser at: http://localhost:8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        panic(err)
    }
}
