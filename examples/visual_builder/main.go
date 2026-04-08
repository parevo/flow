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
	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/node"
	"github.com/parevo/flow/internal/storage/memory"
)

// ─────────────────────────────────────────────────────────────
// DB — shared in-memory SQLite for all DBQueryNode executions
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
// DBQueryNode — runs a SQL query and injects results into JSON
// ─────────────────────────────────────────────────────────────
//
// Config:
//   - "query"      (string) : SQL SELECT — use {{.field}} from input as bind params
//   - "bind_field" (string) : optional field from input to use as first bind param ($1 / ?)
//   - "result_key" (string) : key to store query results under in output JSON
//   - "mode"       (string) : "one" (single row) or "many" (array). Default: "one"
//
// Example:
//   query: "SELECT name, role, status FROM users WHERE id = ?"
//   bind_field: "user_id"
//   result_key: "user"
//   mode: "one"

type DBQueryNode struct{ db *sql.DB }

func (n *DBQueryNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	query, _ := config["query"].(string)
	bindField, _ := config["bind_field"].(string)
	resultKey, _ := config["result_key"].(string)
	if resultKey == "" {
		resultKey = "db_result"
	}
	mode, _ := config["mode"].(string)
	if mode == "" {
		mode = "one"
	}

	// Parse input
	var inMap map[string]interface{}
	_ = json.Unmarshal([]byte(input), &inMap)
	if inMap == nil {
		inMap = map[string]interface{}{}
	}

	// Build bind params
	var args []interface{}
	if bindField != "" {
		if v, ok := inMap[bindField]; ok {
			args = append(args, v)
		}
	}

	fmt.Printf("\n🗄️  [DB QUERY] SQL: %s | Params: %v\n", query, args)

	rows, err := n.db.Query(query, args...)
	if err != nil {
		return "", fmt.Errorf("db_query: SQL error: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		row := map[string]interface{}{}
		for i, col := range cols {
			b, ok := vals[i].([]byte)
			if ok {
				row[col] = string(b)
			} else {
				row[col] = vals[i]
			}
		}
		results = append(results, row)
	}

	// Merge into output
	out := make(map[string]interface{})
	for k, v := range inMap {
		out[k] = v
	}

	if mode == "one" {
		if len(results) > 0 {
			out[resultKey] = results[0]
			// Flatten result fields into top-level for easy condition access
			for k, v := range results[0] {
				out[k] = v
			}
		} else {
			out[resultKey] = nil
			out["db_found"] = false
		}
	} else {
		out[resultKey] = results
		out["db_count"] = len(results)
	}

	b, _ := json.Marshal(out)
	fmt.Printf("   → Result: %s\n", string(b))
	return string(b), nil
}

func (n *DBQueryNode) Validate(config map[string]interface{}) error {
	if _, ok := config["query"].(string); !ok {
		return fmt.Errorf("db_query: missing 'query' in config")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// DBWriteNode — INSERT / UPDATE / DELETE
// ─────────────────────────────────────────────────────────────
//
// Config:
//   - "query"       (string)   : SQL with ? placeholders
//   - "bind_fields" ([]string) : list of input field names to bind in order

type DBWriteNode struct{ db *sql.DB }

func (n *DBWriteNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	query, _ := config["query"].(string)
	var inMap map[string]interface{}
	_ = json.Unmarshal([]byte(input), &inMap)
	if inMap == nil {
		inMap = map[string]interface{}{}
	}

	var args []interface{}
	// Handle both string (comma-separated from UI) and array formats
	if bfStr, ok := config["bind_fields"].(string); ok {
		// UI sends comma-separated string
		fields := strings.Split(bfStr, ",")
		for _, f := range fields {
			key := strings.TrimSpace(f)
			if key != "" {
				args = append(args, inMap[key])
			}
		}
	} else if bf, ok := config["bind_fields"].([]interface{}); ok {
		// Array format (direct API usage)
		for _, f := range bf {
			key := fmt.Sprintf("%v", f)
			args = append(args, inMap[key])
		}
	}

	fmt.Printf("\n✏️  [DB WRITE] SQL: %s | Params: %v\n", query, args)

	res, err := n.db.Exec(query, args...)
	if err != nil {
		return "", fmt.Errorf("db_write: SQL error: %w", err)
	}
	affected, _ := res.RowsAffected()
	lastID, _ := res.LastInsertId()

	out := make(map[string]interface{})
	for k, v := range inMap {
		out[k] = v
	}
	out["rows_affected"] = affected
	out["last_insert_id"] = lastID
	delete(out, "branch")

	b, _ := json.Marshal(out)
	return string(b), nil
}

func (n *DBWriteNode) Validate(config map[string]interface{}) error {
	if _, ok := config["query"].(string); !ok {
		return fmt.Errorf("db_write: missing 'query' in config")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// Supporting UI nodes (carry-through, strip branch tag)
// ─────────────────────────────────────────────────────────────

type LogNode struct{}

func (n *LogNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	msg, _ := config["message"].(string)
	fmt.Printf("\n📝 [LOG] %s\n", msg)
	if input == "" {
		input = "{}"
	}
	var m map[string]interface{}
	json.Unmarshal([]byte(input), &m)
	delete(m, "branch")
	out, _ := json.Marshal(m)
	return string(out), nil
}
func (n *LogNode) Validate(_ map[string]interface{}) error { return nil }

type ProgressNode struct{}

func (n *ProgressNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	stepName, _ := config["step_name"].(string)
	progress, _ := config["progress"].(string)
	fmt.Printf("\n📊 [PROGRESS] %s → %s\n", stepName, progress)
	if input == "" {
		input = "{}"
	}
	var m map[string]interface{}
	json.Unmarshal([]byte(input), &m)
	delete(m, "branch")
	out, _ := json.Marshal(m)
	return string(out), nil
}
func (n *ProgressNode) Validate(_ map[string]interface{}) error { return nil }

// ─────────────────────────────────────────────────────────────
// HTTP request/response types
// ─────────────────────────────────────────────────────────────

type UINode struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}
type UIEdge struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Condition string `json:"condition"`
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

// ─────────────────────────────────────────────────────────────
// main
// ─────────────────────────────────────────────────────────────

func main() {
	initDB()

	memStore := memory.NewMemoryStorage()
	registry := engine.NewRegistry()

	// Built-in nodes
	registry.Register("log", &LogNode{})
	registry.Register("progress", &ProgressNode{})
	registry.Register("wait", &node.WaitNode{})
	registry.Register("http", node.NewHTTPNode())
	registry.Register("condition", &node.ConditionNode{})
	registry.Register("signal", &node.SignalNode{})
	registry.Register("switch", &node.SwitchNode{})
	registry.Register("set_variable", &node.SetVariableNode{})
	registry.Register("transform", &node.TransformNode{})
	registry.Register("notify", node.NewNotifyNode())
	registry.Register("log_node", node.NewLogNode())

	// DB nodes
	registry.Register("db_query", &DBQueryNode{db: sharedDB})
	registry.Register("db_write", &DBWriteNode{db: sharedDB})

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := engine.NewEngine(memStore, registry).WithLogger(logger)
	ctx := context.Background()

	w := engine.NewWorker("visual-worker-1", eng, registry, 100*time.Millisecond)
	w.SetNamespace("visual-ns")
	go w.Start(ctx)

	// Serve UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// Run a workflow from the UI canvas
	http.HandleFunc("/api/run", func(wr http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(wr, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload UIWorkflowPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(wr, "Invalid JSON", http.StatusBadRequest)
			return
		}

		wfID := fmt.Sprintf("visual-wf-%d", time.Now().UnixMilli())
		wf := &models.Workflow{ID: wfID, Name: "Visual Workflow"}

		for _, un := range payload.Nodes {
			wf.Nodes = append(wf.Nodes, models.Node{ID: un.ID, Type: un.Name, Config: un.Data})
		}
		for _, ue := range payload.Edges {
			wf.Edges = append(wf.Edges, models.Edge{SourceID: ue.Source, TargetID: ue.Target, Condition: ue.Condition})
		}

		memStore.SaveWorkflow(ctx, "visual-ns", wf)

		// Initial input: allow injecting via query string e.g. ?user_id=2
		initInput := map[string]interface{}{}
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				initInput[k] = v[0]
			}
		}
		inputBytes, _ := json.Marshal(initInput)

		execID, err := eng.StartWorkflow(ctx, "visual-ns", wfID, string(inputBytes))
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("\n🚀 Workflow started! ExecID: %s | Input: %s\n", execID, string(inputBytes))
		wr.Header().Set("Content-Type", "application/json")
		wr.Write([]byte(`{"status":"success","execution_id":"` + execID + `"}`))
	})

	// Signal
	http.HandleFunc("/api/signal", func(wr http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(wr, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var sp SignalPayload
		if err := json.Unmarshal(body, &sp); err != nil {
			http.Error(wr, "Invalid JSON", http.StatusBadRequest)
			return
		}
		err := eng.SignalExecution(ctx, "visual-ns", sp.ExecutionID, sp.NodeID, sp.Data)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("\n⚡ SIGNAL → Exec: %s | Node: %s\n", sp.ExecutionID, sp.NodeID)
		wr.Header().Set("Content-Type", "application/json")
		wr.Write([]byte(`{"status":"success"}`))
	})

	// Status
	http.HandleFunc("/api/status", func(wr http.ResponseWriter, r *http.Request) {
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
		result := map[string]map[string]string{}
		for _, s := range steps {
			result[s.NodeID] = map[string]string{
				"status": string(s.Status),
				"error":  s.Error,
				"output": truncate(s.Output, 200),
			}
		}
		b, _ := json.Marshal(result)
		wr.Header().Set("Content-Type", "application/json")
		wr.Write(b)
	})

	// DB inspect endpoint (for debugging)
	http.HandleFunc("/api/db", func(wr http.ResponseWriter, r *http.Request) {
		table := r.URL.Query().Get("table")
		if table == "" || strings.ContainsAny(table, " ;'\"") {
			http.Error(wr, "invalid table", http.StatusBadRequest)
			return
		}
		rows, err := sharedDB.Query("SELECT * FROM " + table)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		var records []map[string]interface{}
		for rows.Next() {
			vals := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			rows.Scan(ptrs...)
			row := map[string]interface{}{}
			for i, col := range cols {
				b, ok := vals[i].([]byte)
				if ok {
					row[col] = string(b)
				} else {
					row[col] = vals[i]
				}
			}
			records = append(records, row)
		}
		b, _ := json.Marshal(map[string]interface{}{"table": table, "rows": records})
		wr.Header().Set("Content-Type", "application/json")
		wr.Write(b)
	})

	fmt.Println("🌟 Parevo Flow Visual Studio (SQL Edition) is running!")
	fmt.Println("👉 Open: http://localhost:8080")
	fmt.Println("🗄️  DB Inspect: http://localhost:8080/api/db?table=users")
	fmt.Println("📋 Tables: users | badges | documents | invoices | audit_log")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
