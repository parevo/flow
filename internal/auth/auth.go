package auth

import "context"

// AuthProvider is the interface for authorization
// Implement this with YOUR auth system - we don't care how
type AuthProvider interface {
	// CheckAccess checks if action is allowed on resource
	// Use context to pass whatever you need (user_id, tenant_id, slug, JWT, etc.)
	//
	// Examples:
	//   CheckAccess(ctx, "workflow:my-flow", "execute")
	//   CheckAccess(ctx, "workflow:my-flow", "edit")
	//   CheckAccess(ctx, "execution:abc-123", "view")
	//
	// Return nil if allowed, error if denied
	CheckAccess(ctx context.Context, resource string, action string) error
}

// NoAuth allows everything - for dev/testing
type NoAuth struct{}

func (n *NoAuth) CheckAccess(ctx context.Context, resource string, action string) error {
	return nil
}
