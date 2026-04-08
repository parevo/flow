package node

import (
	"context"
	"encoding/json"
	"fmt"
)

// SwitchNode routes execution to one of N branches based on the value of a field.
// Unlike ConditionNode (true/false only), SwitchNode supports any number of cases.
//
// Config:
//   - "variable" (string, required): the field name to read from input JSON
//   - "cases"    (map, required)   : map of expected value -> branch label
//                                    Example: {"premium": "branch-premium", "free": "branch-free"}
//   - "default"  (string, optional): branch label when no case matches (defaults to "default")
//
// Output: JSON with "branch" set to the matched case label.
//         The DAG engine uses this to follow only the matching edge.
//
// Usage in workflow edges:
//   { "sourceId": "switch-node", "targetId": "handle-premium", "condition": "branch-premium" }
//   { "sourceId": "switch-node", "targetId": "handle-free",    "condition": "branch-free"    }
//   { "sourceId": "switch-node", "targetId": "handle-default", "condition": "default"        }
type SwitchNode struct{}

func (n *SwitchNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	var inMap map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inMap); err != nil {
		return "", fmt.Errorf("switch: invalid input JSON: %w", err)
	}

	variable, _ := config["variable"].(string)
	if variable == "" {
		return "", fmt.Errorf("switch: 'variable' is required")
	}

	casesRaw, ok := config["cases"]
	if !ok {
		return "", fmt.Errorf("switch: 'cases' is required")
	}
	cases, ok := casesRaw.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("switch: 'cases' must be an object")
	}

	defaultBranch := "default"
	if d, ok := config["default"].(string); ok && d != "" {
		defaultBranch = d
	}

	// Get input value
	rawVal, exists := inMap[variable]
	if !exists {
		return buildSwitchOutput(inMap, defaultBranch), nil
	}

	valStr := fmt.Sprintf("%v", rawVal)

	// Match against cases
	for caseVal, branchRaw := range cases {
		if caseVal == valStr {
			branch := fmt.Sprintf("%v", branchRaw)
			return buildSwitchOutput(inMap, branch), nil
		}
	}

	// No match → default branch
	return buildSwitchOutput(inMap, defaultBranch), nil
}

func buildSwitchOutput(inMap map[string]interface{}, branch string) string {
	// Merge the branch tag into the existing data so downstream nodes can use original fields  
	out := make(map[string]interface{})
	for k, v := range inMap {
		out[k] = v
	}
	out["branch"] = branch
	b, _ := json.Marshal(out)
	return string(b)
}

func (n *SwitchNode) Validate(config map[string]interface{}) error {
	if _, ok := config["variable"].(string); !ok {
		return fmt.Errorf("switch: missing 'variable' in config")
	}
	if _, ok := config["cases"]; !ok {
		return fmt.Errorf("switch: missing 'cases' in config")
	}
	return nil
}
