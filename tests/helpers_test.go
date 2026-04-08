package tests

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockNode returns its name and input combined
type MockNode struct {
	Name string
}
func (m *MockNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	fmt.Printf("[TEST] Executing %s with input: %s\n", m.Name, input)
	return fmt.Sprintf("%s_DONE", m.Name), nil
}
func (m *MockNode) Validate(config map[string]interface{}) error { return nil }

// FailureNode fails for a set number of attempts
type FailureNode struct {
	FailUntil int
	mu        sync.Mutex
	attempts  int
}
func (f *FailureNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.attempts < f.FailUntil {
		return "", fmt.Errorf("SIMULATED_FAILURE_ATTEMPT_%d", f.attempts)
	}
	return "RECOVERED_SUCCESS", nil
}
func (f *FailureNode) Validate(config map[string]interface{}) error { return nil }

// CountingNode increments a global counter
type CountingNode struct {
	Counter *int
	mu      *sync.Mutex
}
func (c *CountingNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	*c.Counter++
	return fmt.Sprintf("COUNT_%d", *c.Counter), nil
}
func (c *CountingNode) Validate(config map[string]interface{}) error { return nil }

// SleepNode delays execution
type SleepNode struct {
	Duration time.Duration
}
func (s *SleepNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	time.Sleep(s.Duration)
	return "SLEEP_DONE", nil
}
func (s *SleepNode) Validate(config map[string]interface{}) error { return nil }

// DataNode preserves JSON input and merges its result
type DataNode struct{}
func (d *DataNode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	fmt.Printf("[TEST] DataNode passing input: %s\n", input)
	return input, nil
}
func (d *DataNode) Validate(config map[string]interface{}) error { return nil }
