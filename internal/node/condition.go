package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ConditionNode evaluates a condition and returns a branch ("true" or "false").
//
// Config:
//   - "variable"  (string) : key to read from the input JSON
//   - "operator"  (string) : ==, !=, >, <, >=, <=, contains, not_contains
//   - "value"     (any)    : value to compare against
//
// Output: {"branch": "true"} or {"branch": "false"}
// Original input fields are preserved alongside "branch" so downstream nodes
// can still read them without needing a transform step.
type ConditionNode struct{}

func (n *ConditionNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	var inMap map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inMap); err != nil {
		return "", fmt.Errorf("failed to parse input JSON: %w", err)
	}

	variable, _ := config["variable"].(string)
	operator, _ := config["operator"].(string)
	value, _ := config["value"]

	inputValue, exists := inMap[variable]
	if !exists {
		return buildCondOutput(inMap, "false"), nil
	}

	aStr := fmt.Sprintf("%v", inputValue)
	bStr := fmt.Sprintf("%v", value)

	result := false
	switch operator {
	case "==", "equal":
		result = aStr == bStr
	case "!=", "not_equal":
		result = aStr != bStr
	case ">":
		var a, b float64
		fmt.Sscanf(aStr, "%f", &a)
		fmt.Sscanf(bStr, "%f", &b)
		result = a > b
	case "<":
		var a, b float64
		fmt.Sscanf(aStr, "%f", &a)
		fmt.Sscanf(bStr, "%f", &b)
		result = a < b
	case ">=":
		var a, b float64
		fmt.Sscanf(aStr, "%f", &a)
		fmt.Sscanf(bStr, "%f", &b)
		result = a >= b
	case "<=":
		var a, b float64
		fmt.Sscanf(aStr, "%f", &a)
		fmt.Sscanf(bStr, "%f", &b)
		result = a <= b
	case "contains":
		result = strings.Contains(aStr, bStr)
	case "not_contains":
		result = !strings.Contains(aStr, bStr)
	default:
		return "", fmt.Errorf("unsupported operator: %s", operator)
	}

	branch := "false"
	if result {
		branch = "true"
	}
	return buildCondOutput(inMap, branch), nil
}

// buildCondOutput merges the branch tag back into the input data so
// downstream nodes still have access to original fields.
func buildCondOutput(inMap map[string]interface{}, branch string) string {
	out := make(map[string]interface{})
	for k, v := range inMap {
		out[k] = v
	}
	out["branch"] = branch
	b, _ := json.Marshal(out)
	return string(b)
}

func containsStr(a, b string) bool {
	return strings.Contains(a, b)
}

func (n *ConditionNode) Validate(config map[string]interface{}) error {
	if _, ok := config["variable"].(string); !ok {
		return fmt.Errorf("missing 'variable' in condition config")
	}
	if _, ok := config["operator"].(string); !ok {
		return fmt.Errorf("missing 'operator' in condition config")
	}
	if _, ok := config["value"]; !ok {
		return fmt.Errorf("missing 'value' in condition config")
	}
	return nil
}
