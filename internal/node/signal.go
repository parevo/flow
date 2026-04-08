package node

import (
	"context"
	"fmt"
)

// SignalNode represents a point in the workflow that waits for external input (Signal).
// It returns a special status that causes the worker to pause execution for this path.
type SignalNode struct{}

func (s *SignalNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	// The SignalNode essentially does nothing during its initial execution.
	// It is intended to be moved into TaskWaiting by the worker or engine.
	// However, since our worker currently handles TaskCompleted/TaskFailed,
	// we need a mechanism to tell the worker: "Stop here, wait for signal".
	
	// We return a specific error that the worker will interpret as "Wait for Signal".
	return "", fmt.Errorf("SIGNAL_WAIT")
}

func (s *SignalNode) Validate(config map[string]interface{}) error {
	// SignalNode might not need specific config, but could have a 'timeout' in the future.
	return nil
}
