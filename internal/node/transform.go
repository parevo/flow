package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
)

// TransformNode reshapes or filters JSON data between two nodes.
//
// Config:
//   - "mapping" (map[string]string): maps output keys to Go template expressions
//     Example: {"fullName": "{{.first_name}} {{.last_name}}", "email": "{{.email}}"}
//
// Input:  JSON object
// Output: new JSON object with keys defined by mapping
type TransformNode struct{}

func (n *TransformNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	// Parse input
	var inMap map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inMap); err != nil {
		return "", fmt.Errorf("transform: invalid input JSON: %w", err)
	}

	mappingRaw, ok := config["mapping"]
	if !ok {
		// No mapping defined → pass through unchanged
		return input, nil
	}

	mappingMap, ok := mappingRaw.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("transform: 'mapping' must be an object of key->template pairs")
	}

	out := make(map[string]interface{})
	for outKey, tmplRaw := range mappingMap {
		tmplStr, ok := tmplRaw.(string)
		if !ok {
			return "", fmt.Errorf("transform: mapping value for '%s' must be a string template", outKey)
		}
		t, err := template.New(outKey).Parse(tmplStr)
		if err != nil {
			return "", fmt.Errorf("transform: invalid template for '%s': %w", outKey, err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, inMap); err != nil {
			return "", fmt.Errorf("transform: template execution failed for '%s': %w", outKey, err)
		}
		out[outKey] = buf.String()
	}

	b, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("transform: failed to marshal output: %w", err)
	}
	return string(b), nil
}

func (n *TransformNode) Validate(config map[string]interface{}) error {
	if _, ok := config["mapping"]; !ok {
		return fmt.Errorf("transform: missing 'mapping' in config")
	}
	return nil
}
