package node

import (
	"context"
	"fmt"
	"log"
	"time"
)

type WaitNode struct{}

func (w *WaitNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	durationStr, ok := config["duration"].(string)
	if !ok {
		return "", fmt.Errorf("duration missing in wait node config")
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return "", fmt.Errorf("invalid duration format: %v", err)
	}

	log.Printf("[WAIT] Pausing for %s...\n", duration)
	
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(duration):
		// Done waiting
	}

	return fmt.Sprintf("Waited for %s", durationStr), nil
}
