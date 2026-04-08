package node

import (
	"context"
	"encoding/json"
	"fmt"
)

// RegistryInterface is the minimal interface FunctionNode needs from Registry
type RegistryInterface interface {
	GetFunction(name string) (func(context.Context, map[string]interface{}) (map[string]interface{}, error), bool)
	ListFunctions() []string
}

// FunctionNode executes user-registered functions
type FunctionNode struct {
	registry RegistryInterface
}

func NewFunctionNode() *FunctionNode {
	return &FunctionNode{}
}

// SetRegistry sets the registry that contains the functions
func (fn *FunctionNode) SetRegistry(registry RegistryInterface) {
	fn.registry = registry
}

// Execute runs the specified function from the registry
func (fn *FunctionNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	funcName, ok := config["function"].(string)
	if !ok {
		return "", fmt.Errorf("function node: missing 'function' name in config")
	}

	if fn.registry == nil {
		return "", fmt.Errorf("function node: registry not set")
	}

	handler, exists := fn.registry.GetFunction(funcName)
	if !exists {
		return "", fmt.Errorf("function node: function '%s' not registered. Available: %v", funcName, fn.registry.ListFunctions())
	}

	// Parse input JSON to map
	var inputMap map[string]interface{}
	if input != "" && input != "{}" {
		if err := json.Unmarshal([]byte(input), &inputMap); err != nil {
			return "", fmt.Errorf("failed to parse input JSON: %w", err)
		}
	} else {
		inputMap = make(map[string]interface{})
	}

	// Call the handler
	output, err := handler(ctx, inputMap)
	if err != nil {
		return "", err
	}

	// Convert output to JSON
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(outputJSON), nil
}

func (fn *FunctionNode) Validate(config map[string]interface{}) error {
	funcName, ok := config["function"].(string)
	if !ok {
		return fmt.Errorf("function node: missing 'function' name in config")
	}

	if fn.registry == nil {
		return fmt.Errorf("function node: registry not set")
	}

	_, exists := fn.registry.GetFunction(funcName)
	if !exists {
		return fmt.Errorf("function node: function '%s' not registered. Available: %v", funcName, fn.registry.ListFunctions())
	}

	return nil
}
