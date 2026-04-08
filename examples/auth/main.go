package main

import (
	"context"
	"fmt"
	"log"

	"github.com/parevo/flow"
)

// Context key types to avoid collisions
type contextKey string

const (
	customerIDKey contextKey = "customer_id"
	userIDKey     contextKey = "user_id"
	roleKey       contextKey = "role"
)

// Example 1: Multi-tenant auth implementation
type MyMultiTenantAuth struct{}

func (a *MyMultiTenantAuth) CheckAccess(ctx context.Context, resource string, action string) error {
	// 1. Get auth info from context
	customerID, _ := ctx.Value(customerIDKey).(string)
	userID, _ := ctx.Value(userIDKey).(string)
	userRole, _ := ctx.Value(roleKey).(string)

	if customerID == "" || userID == "" {
		return fmt.Errorf("unauthorized: missing auth info")
	}

	// 2. Parse resource (format: "workflow:namespace:workflow_id")
	// You can use any format you want!

	// 3. Example: Get workflow metadata (in real app, query from DB/storage)
	// workflow := getWorkflowFromDB(resource)

	// Simulated metadata
	workflowMeta := map[string]interface{}{
		"customer_id": "acme-corp",
		"owner_id":    "user-123",
		"visibility":  "team",
		"team_id":     "eng-team",
	}

	// 4. CRITICAL: Customer isolation check
	if workflowMeta["customer_id"] != customerID {
		return fmt.Errorf("forbidden: customer mismatch")
	}

	// 5. Check permissions
	// Admin can do anything
	if userRole == "admin" {
		return nil
	}

	// Owner can do anything
	if workflowMeta["owner_id"] == userID {
		return nil
	}

	// Team visibility - check if user in team
	if workflowMeta["visibility"] == "team" {
		// In real app: check team membership from DB
		userTeams := []string{"eng-team", "ops-team"} // from DB
		teamID := workflowMeta["team_id"].(string)

		for _, t := range userTeams {
			if t == teamID {
				// Team members can view and execute
				if action == "view" || action == "execute" {
					return nil
				}
			}
		}
	}

	// Organization visibility - everyone in customer can view/execute
	if workflowMeta["visibility"] == "organization" {
		if action == "view" || action == "execute" {
			return nil
		}
	}

	return fmt.Errorf("forbidden: insufficient permissions")
}

func main() {
	// 1. Setup storage and engine
	storage := flow.NewMemoryStorage()
	registry := flow.NewRegistry()
	engine := flow.NewEngine(storage, registry)

	// 2. Set auth provider
	authProvider := &MyMultiTenantAuth{}
	// engine.SetAuthProvider(authProvider) // We'll implement this

	fmt.Println("✅ Multi-tenant auth example")

	// 3. Create workflow with metadata
	ctx := context.Background()

	// Add auth info to context
	ctx = context.WithValue(ctx, customerIDKey, "acme-corp")
	ctx = context.WithValue(ctx, userIDKey, "user-123")
	ctx = context.WithValue(ctx, roleKey, "user")

	wf := &flow.Workflow{
		ID:      "onboarding-flow",
		Name:    "Employee Onboarding",
		Version: 1,
		Nodes: []flow.Node{
			{ID: "welcome", Type: "log", Config: map[string]interface{}{"message": "Welcome!"}},
		},
		// Store auth metadata
		Metadata: map[string]interface{}{
			"customer_id": "acme-corp",
			"owner_id":    "user-123",
			"visibility":  "team",
			"team_id":     "eng-team",
			"created_by":  "john@acme.com",
		},
	}

	// 4. Register workflow (auth check would happen here)
	if err := engine.RegisterWorkflow(ctx, "default", wf); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ Workflow registered with metadata")

	// 5. Test auth
	fmt.Println("\n--- Testing Auth ---")

	// Owner tries to execute - should pass
	err := authProvider.CheckAccess(ctx, "workflow:default:onboarding-flow", "execute")
	if err == nil {
		fmt.Println("✅ Owner can execute")
	} else {
		fmt.Printf("❌ Owner blocked: %v\n", err)
	}

	// Different user (same customer, same team) tries to execute - should pass
	ctx2 := context.WithValue(context.Background(), customerIDKey, "acme-corp")
	ctx2 = context.WithValue(ctx2, userIDKey, "user-456")
	ctx2 = context.WithValue(ctx2, roleKey, "user")

	err = authProvider.CheckAccess(ctx2, "workflow:default:onboarding-flow", "execute")
	if err == nil {
		fmt.Println("✅ Team member can execute")
	} else {
		fmt.Printf("❌ Team member blocked: %v\n", err)
	}

	// Different customer tries to access - should fail
	ctx3 := context.WithValue(context.Background(), customerIDKey, "other-corp")
	ctx3 = context.WithValue(ctx3, userIDKey, "user-789")
	ctx3 = context.WithValue(ctx3, roleKey, "admin")

	err = authProvider.CheckAccess(ctx3, "workflow:default:onboarding-flow", "execute")
	if err != nil {
		fmt.Println("✅ Different customer blocked (tenant isolation working)")
	} else {
		fmt.Printf("❌ SECURITY ISSUE: Different customer can access!\n")
	}
}
