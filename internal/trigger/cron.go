package trigger

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// CronManager handles periodic workflow triggers (Temporal-style Scheduling)
type CronManager struct {
	cron   *cron.Cron
	engine StartableEngine
	logger *slog.Logger
}

type StartableEngine interface {
	StartWorkflow(ctx context.Context, namespace, workflowID string, input string) (string, error)
}

func NewCronManager(engine StartableEngine, logger *slog.Logger) *CronManager {
	return &CronManager{
		cron:   cron.New(cron.WithSeconds()),
		engine: engine,
		logger: logger,
	}
}

// AddSchedule registers a new periodic workflow trigger
func (c *CronManager) AddSchedule(namespace, workflowID, cronExpr, input string) (cron.EntryID, error) {
	id, err := c.cron.AddFunc(cronExpr, func() {
		ctx := context.Background()
		execID, err := c.engine.StartWorkflow(ctx, namespace, workflowID, input)
		if err != nil {
			c.logger.Error("Cron trigger failed to start workflow", 
				"namespace", namespace, 
				"workflow", workflowID, 
				"error", err)
			return
		}
		c.logger.Info("Cron trigger started workflow", 
			"namespace", namespace, 
			"workflow", workflowID, 
			"executionId", execID)
	})

	if err != nil {
		return 0, fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
	}

	return id, nil
}

func (c *CronManager) Start() {
	c.cron.Start()
}

func (c *CronManager) Stop() {
	c.cron.Stop()
}
