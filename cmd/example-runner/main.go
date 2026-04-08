package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/node"
	"github.com/parevo/flow/internal/storage/memory"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize Storage
	store := memory.NewMemoryStorage()

	// 2. Register Node Types
	registry := engine.NewRegistry()
	registry.Register("log", &engine.LogNode{})
	registry.Register("wait", &node.WaitNode{})
	registry.Register("http", node.NewHTTPNode())

	// 3. Initialize Engine & Worker
	eng := engine.NewEngine(store, registry)
	worker := engine.NewWorker("saas-worker-1", eng, registry, 500*time.Millisecond)

	// 4. Load Onboarding Workflow
	wfData, err := os.ReadFile("examples/onboarding.json")
	if err != nil {
		log.Fatalf("Failed to read workflow file: %v", err)
	}

	var wf models.Workflow
	if err := json.Unmarshal(wfData, &wf); err != nil {
		log.Fatalf("Failed to parse workflow: %v", err)
	}

	// 5. Save Workflow to Storage using a Namespace
	namespace := "parevo-saas-customer-1"
	store.SaveWorkflow(ctx, namespace, &wf)
	log.Printf("Successfully loaded workflow: %s into namespace: %s\n", wf.Name, namespace)

	// 6. Start Worker
	go worker.Start(ctx)

	// 7. Start Execution
	log.Println("Starting onboarding flow...")
	execID, err := eng.StartWorkflow(ctx, namespace, wf.ID, `{"email": "ahmet@parevo.cloud"}`)
	if err != nil {
		log.Fatalf("Failed to start workflow: %v", err)
	}

	log.Printf("Execution started: %s\n", execID)

	time.Sleep(10 * time.Second)
	log.Println("Example runner finished.")
}
