package node

import (
	"context"
	"encoding/json"
	"fmt"
)

// ConditionNode evaluates a simple condition and returns a branch
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
		return `{"branch": "false", "error": "variable not found"}`, nil
	}

	result := false
	switch operator {
	case "==", "equal":
		result = fmt.Sprintf("%v", inputValue) == fmt.Sprintf("%v", value)
	case "!=", "not_equal":
		result = fmt.Sprintf("%v", inputValue) != fmt.Sprintf("%v", value)
	default:
		return "", fmt.Errorf("unsupported operator: %s", operator)
	}

	branch := "false"
	if result {
		branch = "true"
	}

	return fmt.Sprintf(`{"branch": "%s"}`, branch), nil
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
