package node

import (
	"context"
	"encoding/json"
	"fmt"
)

// SetVariableNode injects or overwrites keys on the flowing JSON context.
// It's the workflow equivalent of `ctx["key"] = value` — no external calls,
// just pure data enrichment so downstream nodes always have what they need.
//
// Config:
//   - "variables" (map, required): key→value pairs to write into the JSON context
//     Values can be any JSON-compatible type: string, number, bool, object.
//
// Example config:
//
//	{
//	  "variables": {
//	    "status":   "pending",
//	    "maxRetry": 3,
//	    "meta":     {"source": "api"}
//	  }
//	}
//
// Input:  {"userId": "abc"}
// Output: {"userId": "abc", "status": "pending", "maxRetry": 3, "meta": {"source": "api"}}
type SetVariableNode struct{}

func (n *SetVariableNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	// Start from existing context
	out := make(map[string]interface{})
	if input != "" && input != "null" {
		if err := json.Unmarshal([]byte(input), &out); err != nil {
			return "", fmt.Errorf("set_variable: invalid input JSON: %w", err)
		}
	}

	variables, ok := config["variables"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("set_variable: 'variables' must be an object")
	}

	// Merge variables — config values overwrite input values for same keys
	for k, v := range variables {
		out[k] = v
	}

	b, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("set_variable: marshal error: %w", err)
	}
	return string(b), nil
}

func (n *SetVariableNode) Validate(config map[string]interface{}) error {
	if _, ok := config["variables"]; !ok {
		return fmt.Errorf("set_variable: missing 'variables' in config")
	}
	if _, ok := config["variables"].(map[string]interface{}); !ok {
		return fmt.Errorf("set_variable: 'variables' must be an object")
	}
	return nil
}
