package node

import (
	"context"
	"fmt"
	"sync"
)

// HandlerFunc is a function type for custom business logic
// This allows users to register inline functions without creating separate node structs
type HandlerFunc func(ctx context.Context, config map[string]interface{}, input string) (string, error)

// FunctionNode allows registering custom business logic as functions
// This is the most flexible pattern - no need to create a new node type for each business logic!
//
// Usage:
//   funcNode := node.NewFunctionNode()
//   funcNode.Register("createCalendar", myCalendarFunction)
//   funcNode.Register("processInvoice", myInvoiceFunction)
//   registry.Register("function", funcNode)
//
//   wf.AddNode("step1", "function").WithConfig("function", "createCalendar")
type FunctionNode struct {
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
}

func NewFunctionNode() *FunctionNode {
	return &FunctionNode{
		handlers: make(map[string]HandlerFunc),
	}
}

// Register adds a named function handler
func (fn *FunctionNode) Register(name string, handler HandlerFunc) {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	fn.handlers[name] = handler
}

// Unregister removes a function handler
func (fn *FunctionNode) Unregister(name string) {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	delete(fn.handlers, name)
}

// List returns all registered function names
func (fn *FunctionNode) List() []string {
	fn.mu.RLock()
	defer fn.mu.RUnlock()
	
	names := make([]string, 0, len(fn.handlers))
	for name := range fn.handlers {
		names = append(names, name)
	}
	return names
}

// Execute runs the specified function
func (fn *FunctionNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	funcName, ok := config["function"].(string)
	if !ok {
		return "", fmt.Errorf("function node: missing 'function' name in config")
	}
	
	fn.mu.RLock()
	handler, ok := fn.handlers[funcName]
	fn.mu.RUnlock()
	
	if !ok {
		return "", fmt.Errorf("function node: function '%s' not registered. Available: %v", funcName, fn.List())
	}
	
	return handler(ctx, config, input)
}

func (fn *FunctionNode) Validate(config map[string]interface{}) error {
	funcName, ok := config["function"].(string)
	if !ok {
		return fmt.Errorf("function node: missing 'function' name in config")
	}
	
	fn.mu.RLock()
	_, exists := fn.handlers[funcName]
	fn.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("function node: function '%s' not registered. Available: %v", funcName, fn.List())
	}
	
	return nil
}
