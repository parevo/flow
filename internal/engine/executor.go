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

// Registry holds the map of available node executors
type Registry struct {
	executors map[string]NodeExecutor
}

func NewRegistry() *Registry {
	return &Registry{executors: make(map[string]NodeExecutor)}
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

// Register built-in node types via node.NewLogNode(), node.NewNotifyNode(), etc.
// See internal/node/ for all available executors.
