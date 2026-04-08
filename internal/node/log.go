package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// LogNode writes a structured audit log entry and passes input downstream unchanged.
//
// Config:
//   - "message"  (string, required) : log message
//   - "level"    (string, optional) : "info" | "warn" | "error" (default: "info")
//   - "fields"   (map, optional)    : extra fixed fields to include in the log entry
//
// Output: passes input JSON through unchanged (non-transforming, side-effect node)
type LogNode struct {
	logger *slog.Logger
}

func NewLogNode() *LogNode {
	return &LogNode{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (n *LogNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	message, _ := config["message"].(string)
	if message == "" {
		message = "workflow step executed"
	}
	level, _ := config["level"].(string)

	// Parse input for structured context
	var inMap map[string]interface{}
	json.Unmarshal([]byte(input), &inMap)

	// Build slog args
	args := []interface{}{"input", inMap}

	// Add optional extra fields from config
	if fields, ok := config["fields"].(map[string]interface{}); ok {
		for k, v := range fields {
			args = append(args, k, v)
		}
	}

	switch level {
	case "warn", "warning":
		n.logger.Warn(message, args...)
	case "error":
		n.logger.Error(message, args...)
	default:
		n.logger.Info(message, args...)
	}

	// Pass input through unchanged
	if input == "" {
		return "{}", nil
	}
	return input, nil
}

func (n *LogNode) Validate(config map[string]interface{}) error {
	if _, ok := config["message"].(string); !ok {
		return fmt.Errorf("log: missing 'message' in config")
	}
	return nil
}
