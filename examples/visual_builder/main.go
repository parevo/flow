package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/parevo/flow"
)

// ─────────────────────────────────────────────────────────────
// DB — shared in-memory SQLite for demo purposes
// ─────────────────────────────────────────────────────────────

var sharedDB *sql.DB

func initDB() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		panic("failed to open in-memory SQLite: " + err.Error())
	}
	sharedDB = db

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		name TEXT, email TEXT,
		role TEXT DEFAULT 'employee',
		department TEXT,
		status TEXT DEFAULT 'active',
		salary REAL,
		created_at TEXT
	);
	CREATE TABLE IF NOT EXISTS badges (
		id INTEGER PRIMARY KEY,
		user_id INTEGER,
		name TEXT,
		status TEXT DEFAULT 'pending',
		issued_at TEXT,
		expires_at TEXT
	);
	CREATE TABLE IF NOT EXISTS documents (
		id INTEGER PRIMARY KEY,
		user_id INTEGER,
		type TEXT,
		file_url TEXT,
		verified INTEGER DEFAULT 0,
		uploaded_at TEXT
	);
	CREATE TABLE IF NOT EXISTS invoices (
		id INTEGER PRIMARY KEY,
		user_id INTEGER,
		amount REAL,
		status TEXT DEFAULT 'pending',
		due_date TEXT,
		paid_at TEXT
	);
	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		action TEXT,
		detail TEXT,
		created_at TEXT
	);

	INSERT OR IGNORE INTO users VALUES (1,'Ahmet Bilgay','ahmet@parevo.com','admin','Engineering','active',75000,'2023-01-15');
	INSERT OR IGNORE INTO users VALUES (2,'Zeynep Yılmaz','zeynep@parevo.com','employee','HR','active',55000,'2023-03-20');
	INSERT OR IGNORE INTO users VALUES (3,'Mert Kaya','mert@parevo.com','employee','Sales','suspended',45000,'2022-11-10');
	INSERT OR IGNORE INTO users VALUES (4,'Elif Şahin','elif@parevo.com','employee','Finance','active',60000,'2024-01-05');

	INSERT OR IGNORE INTO badges VALUES (1,1,'Field Expert','active','2024-01-01','2025-01-01');
	INSERT OR IGNORE INTO badges VALUES (2,1,'OHS Certified','active','2024-02-01','2025-02-01');
	INSERT OR IGNORE INTO badges VALUES (3,2,'HR Specialist','pending',NULL,NULL);
	INSERT OR IGNORE INTO badges VALUES (4,3,'Sales Pro','revoked','2023-06-01','2024-06-01');

	INSERT OR IGNORE INTO documents VALUES (1,2,'drivers_license','https://s3.ex/dl_zeynep.pdf',0,'2024-03-10');
	INSERT OR IGNORE INTO documents VALUES (2,2,'isg_certificate','https://s3.ex/isg_zeynep.pdf',1,'2024-02-20');
	INSERT OR IGNORE INTO documents VALUES (3,4,'financial_cert','https://s3.ex/fin_elif.pdf',0,'2024-04-01');

	INSERT OR IGNORE INTO invoices VALUES (1,1,12500.00,'paid','2024-02-28','2024-02-25');
	INSERT OR IGNORE INTO invoices VALUES (2,2,3200.00,'pending','2024-05-01',NULL);
	INSERT OR IGNORE INTO invoices VALUES (3,3,8900.00,'overdue','2024-01-15',NULL);
	INSERT OR IGNORE INTO invoices VALUES (4,4,5600.00,'pending','2024-05-15',NULL);
	`
	if _, err := db.Exec(schema); err != nil {
		panic("DB schema error: " + err.Error())
	}
	fmt.Println("📦 In-memory SQLite database initialized with seed data.")
}

// ─────────────────────────────────────────────────────────────
// Custom Node: DBQueryNode
// ─────────────────────────────────────────────────────────────

type DBQueryNode struct{ db *sql.DB }

func (n *DBQueryNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	query, _ := config["query"].(string)
	if query == "" {
		return "", fmt.Errorf("DBQueryNode: missing 'query' in config")
	}

	var inputMap map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inputMap); err != nil {
		return "", fmt.Errorf("failed to parse input JSON: %w", err)
	}

	bindField, _ := config["bind_field"].(string)
	resultKey, _ := config["result_key"].(string)
	mode, _ := config["mode"].(string)
	if mode == "" {
		mode = "one"
	}

	var args []interface{}
	if bindField != "" {
		if val, ok := inputMap[bindField]; ok {
			args = append(args, val)
		}
	}

	rows, err := n.db.Query(query, args...)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}

		rowMap := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	outputMap := inputMap
	if mode == "one" && len(results) > 0 {
		outputMap[resultKey] = results[0]
	} else {
		outputMap[resultKey] = results
	}

	outputJSON, err := json.Marshal(outputMap)
	if err != nil {
		return "", err
	}
	return string(outputJSON), nil
}

func (n *DBQueryNode) Validate(config map[string]interface{}) error {
	if _, ok := config["query"].(string); !ok {
		return fmt.Errorf("missing required config: query")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// Custom Node: DBWriteNode
// ─────────────────────────────────────────────────────────────

type DBWriteNode struct{ db *sql.DB }

func (n *DBWriteNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	statement, _ := config["statement"].(string)
	if statement == "" {
		return "", fmt.Errorf("DBWriteNode: missing 'statement' in config")
	}

	var inputMap map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inputMap); err != nil {
		return "", err
	}

	bindFields, _ := config["bind_fields"].([]interface{})
	var args []interface{}
	for _, f := range bindFields {
		field := fmt.Sprintf("%v", f)
		if val, ok := inputMap[field]; ok {
			args = append(args, val)
		}
	}

	result, err := n.db.Exec(statement, args...)
	if err != nil {
		return "", fmt.Errorf("write error: %w", err)
	}

	affected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()

	inputMap["rows_affected"] = affected
	inputMap["last_insert_id"] = lastID

	outputJSON, _ := json.Marshal(inputMap)
	return string(outputJSON), nil
}

func (n *DBWriteNode) Validate(config map[string]interface{}) error {
	if _, ok := config["statement"].(string); !ok {
		return fmt.Errorf("missing required config: statement")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// Custom Node: LogNode
// ─────────────────────────────────────────────────────────────

type LogNode struct{}

func (n *LogNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	msg, _ := config["message"].(string)
	fmt.Printf("📝 LOG: %s\n", msg)
	return input, nil
}

func (n *LogNode) Validate(config map[string]interface{}) error { return nil }

// ─────────────────────────────────────────────────────────────
// Custom Node: ProgressNode (for UI updates)
// ─────────────────────────────────────────────────────────────

type ProgressNode struct{}

func (n *ProgressNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	message, _ := config["message"].(string)
	fmt.Printf("⏳ Progress: %s\n", message)
	time.Sleep(500 * time.Millisecond)
	return input, nil
}

func (n *ProgressNode) Validate(config map[string]interface{}) error { return nil }

// ─────────────────────────────────────────────────────────────
// API Types for UI
// ─────────────────────────────────────────────────────────────

type UINode struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}
type UIEdge struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Condition string `json:"condition,omitempty"`
}
type UIWorkflowPayload struct {
	Nodes []UINode `json:"nodes"`
	Edges []UIEdge `json:"edges"`
}
type SignalPayload struct {
	ExecutionID string                 `json:"execution_id"`
	NodeID      string                 `json:"node_id"`
	Data        map[string]interface{} `json:"data"`
}

// ─────────────────────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────────────────────

func main() {
	// 1. Initialize database
	initDB()

	// 2. Setup Parevo Flow
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	// 3. Register custom nodes
	registry.Register("db_query", &DBQueryNode{db: sharedDB})
	registry.Register("db_write", &DBWriteNode{db: sharedDB})
	registry.Register("log", &LogNode{})
	registry.Register("progress", &ProgressNode{})

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	engine.WithLogger(logger)

	ctx := context.Background()

	// 4. Start worker
	go engine.StartWorker(ctx, "visual-builder", "worker-ui-1")

	fmt.Println("🚀 Parevo Flow Visual Builder started!")
	fmt.Println("📍 Open: http://localhost:3000")
	fmt.Println("─────────────────────────────────────")

	// 5. HTTP Handlers

	// Serve UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	// Create and execute workflow from UI
	http.HandleFunc("/api/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var payload UIWorkflowPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Convert UI nodes to flow.Node
		var nodes []flow.Node
		for _, uiNode := range payload.Nodes {
			nodeType, _ := uiNode.Data["type"].(string)
			config := make(map[string]interface{})
			for k, v := range uiNode.Data {
				if k != "type" {
					config[k] = v
				}
			}
			nodes = append(nodes, flow.Node{
				ID:     uiNode.ID,
				Type:   nodeType,
				Name:   uiNode.Name,
				Config: config,
			})
		}

		// Convert UI edges to flow.Edge
		var edges []flow.Edge
		for _, uiEdge := range payload.Edges {
			edges = append(edges, flow.Edge{
				SourceID:  uiEdge.Source,
				TargetID:  uiEdge.Target,
				Condition: uiEdge.Condition,
			})
		}

		// Create workflow
		wf := &flow.Workflow{
			ID:      fmt.Sprintf("wf-%d", time.Now().Unix()),
			Name:    "Visual Workflow",
			Version: 1,
			Nodes:   nodes,
			Edges:   edges,
		}

		// Register and execute
		if err := engine.RegisterWorkflow(ctx, "visual-builder", wf); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		input := `{"timestamp": "` + time.Now().Format(time.RFC3339) + `"}`
		execID, err := engine.Execute(ctx, "visual-builder", wf.ID, []byte(input))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"execution_id": execID,
			"workflow_id":  wf.ID,
			"status":       "started",
		})
	})

	// Get execution status
	http.HandleFunc("/api/status/", func(w http.ResponseWriter, r *http.Request) {
		execID := strings.TrimPrefix(r.URL.Path, "/api/status/")
		if execID == "" {
			http.Error(w, "Missing execution ID", http.StatusBadRequest)
			return
		}

		exec, err := engine.GetExecution(ctx, "visual-builder", execID)
		if err != nil {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}

		steps, _ := engine.GetExecutionSteps(ctx, "visual-builder", execID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution_id": exec.ID,
			"workflow_id":  exec.WorkflowID,
			"status":       exec.Status,
			"started_at":   exec.StartedAt,
			"finished_at":  exec.FinishedAt,
			"output":       exec.Output,
			"steps":        steps,
		})
	})

	// Send signal to waiting node
	http.HandleFunc("/api/signal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload SignalPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		signalData, _ := json.Marshal(payload.Data)
		err := engine.SignalExecution(ctx, "visual-builder", payload.ExecutionID, payload.NodeID, string(signalData))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "signal_sent"})
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Start server
	fmt.Println("✅ Server ready on http://localhost:3000")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}
}
