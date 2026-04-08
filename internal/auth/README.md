# Auth System - Flexible Authorization

Ultra-minimal, pluggable authorization interface. Bring your own auth!

## Interface

```go
type AuthProvider interface {
    CheckAccess(ctx context.Context, resource string, action string) error
}
```

That's it! Use `context.Context` however you want.

## Quick Start

### 1. No Auth (Default)

```go
engine := flow.NewEngine(storage, registry)
// No auth set = everything allowed
```

### 2. Simple Auth

```go
type MyAuth struct{}

func (a *MyAuth) CheckAccess(ctx context.Context, resource string, action string) error {
    userID := ctx.Value("user_id").(string)
    // Your logic
    return nil
}

engine.SetAuthProvider(&MyAuth{})
```

### 3. Multi-Tenant

```go
func (a *MyAuth) CheckAccess(ctx context.Context, resource string, action string) error {
    customerID := ctx.Value("customer_id").(string)
    userID := ctx.Value("user_id").(string)
    
    // Get workflow
    workflow := getFromDB(resource)
    
    // CRITICAL: Tenant isolation
    if workflow.Metadata["customer_id"] != customerID {
        return errors.New("forbidden")
    }
    
    // Check permissions
    if workflow.Metadata["owner_id"] == userID {
        return nil
    }
    
    return errors.New("forbidden")
}
```

## Store Auth Data

```go
wf := &flow.Workflow{
    ID: "my-workflow",
    Metadata: map[string]interface{}{
        // Use ANY field names - we don't enforce structure!
        "customer_id": "acme-corp",
        "user_id": "user-123",
        "owner": "john@acme.com",
        "team_id": "eng",
        "visibility": "team",
        
        // Or Firebase, Auth0, etc.
        "firebase_uid": "abc123",
        "auth0_org": "org_xyz",
    },
}
```

## Actions

Common actions (define your own):
- `"create"` - Register workflow
- `"view"` - View definition
- `"execute"` - Trigger execution
- `"edit"` - Modify workflow
- `"delete"` - Delete workflow

## Full Example

See `examples/auth/main.go` for complete multi-tenant implementation.

## Key Principles

1. **Fully pluggable** - Use any auth system
2. **No structure enforced** - Use any field names
3. **Context-based** - Pass auth via context
4. **Metadata is flexible** - Store anything
