package node

import (
	"context"
	"fmt"
	"log"
	"time"
)

type WaitNode struct{}

func (w *WaitNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	durationStr, _ := config["duration"].(string)
	duration, _ := time.ParseDuration(durationStr)

	log.Printf("[WAIT] Pausing for %s...\n", duration)
	
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(duration):
		// Done waiting
	}

	return fmt.Sprintf("Waited for %s", durationStr), nil
}

func (w *WaitNode) Validate(config map[string]interface{}) error {
	durationStr, ok := config["duration"].(string)
	if !ok {
		return fmt.Errorf("missing 'duration' in wait config")
	}
	if _, err := time.ParseDuration(durationStr); err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}
	return nil
}
