package engine

import (
	"context"
	"fmt"
)

// NodeExecutor defines the interface for a node's logic
type NodeExecutor interface {
	Execute(ctx context.Context, config map[string]interface{}, input string) (string, error)
	Validate(config map[string]interface{}) error
}

// Registry holds the map of available node executors and functions
type Registry struct {
	executors map[string]NodeExecutor
	functions map[string]func(context.Context, map[string]interface{}) (map[string]interface{}, error)
}

func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[string]NodeExecutor),
		functions: make(map[string]func(context.Context, map[string]interface{}) (map[string]interface{}, error)),
	}
}

func (r *Registry) Register(nodeType string, executor NodeExecutor) {
	r.executors[nodeType] = executor
}

func (r *Registry) Get(nodeType string) (NodeExecutor, error) {
	ex, ok := r.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("node executor not found for type: %s", nodeType)
	}
	return ex, nil
}

// RegisterFunction registers a HandlerFunc as a function node executor
func (r *Registry) RegisterFunction(name string, handler func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)) {
	r.functions[name] = handler
}

// GetFunction retrieves a registered function by name
func (r *Registry) GetFunction(name string) (func(context.Context, map[string]interface{}) (map[string]interface{}, error), bool) {
	handler, exists := r.functions[name]
	return handler, exists
}

// ListFunctions returns all registered function names
func (r *Registry) ListFunctions() []string {
	names := make([]string, 0, len(r.functions))
	for name := range r.functions {
		names = append(names, name)
	}
	return names
}

// Register built-in node types via node.NewLogNode(), node.NewNotifyNode(), etc.
// See internal/node/ for all available executors.
