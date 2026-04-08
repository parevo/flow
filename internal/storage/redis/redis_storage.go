package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/storage"
	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
	crypto *storage.Crypto
}

func NewRedisStorage(addr string, password string, db int) *RedisStorage {
	return &RedisStorage{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (r *RedisStorage) SetEncryption(crypto *storage.Crypto) {
	r.crypto = crypto
}

// Helper keys
func wfKey(ns, id string) string { return fmt.Sprintf("wf:%s:%s", ns, id) }
func execKey(ns, id string) string { return fmt.Sprintf("ex:%s:%s", ns, id) }
func stepKey(ns, id string) string { return fmt.Sprintf("step:%s:%s", ns, id) }
func pendingSet(ns string) string { return fmt.Sprintf("pending:%s", ns) }

func (r *RedisStorage) SaveWorkflow(ctx context.Context, namespace string, wf *models.Workflow) error {
	data, _ := json.Marshal(wf)
	return r.client.Set(ctx, wfKey(namespace, wf.ID), data, 0).Err()
}

func (r *RedisStorage) GetWorkflow(ctx context.Context, namespace string, id string) (*models.Workflow, error) {
	data, err := r.client.Get(ctx, wfKey(namespace, id)).Bytes()
	if err != nil {
		return nil, err
	}
	var wf models.Workflow
	json.Unmarshal(data, &wf)
	return &wf, nil
}

func (r *RedisStorage) ListWorkflows(ctx context.Context, namespace string) ([]*models.Workflow, error) {
	// Simplified: in production, use a dedicated set for listed IDs
	return nil, fmt.Errorf("not implemented in MVP")
}

func (r *RedisStorage) CreateExecution(ctx context.Context, namespace string, exec *models.Execution) error {
	data, _ := json.Marshal(exec)
	return r.client.Set(ctx, execKey(namespace, exec.ID), data, 0).Err()
}

func (r *RedisStorage) GetExecution(ctx context.Context, namespace string, id string) (*models.Execution, error) {
	data, err := r.client.Get(ctx, execKey(namespace, id)).Bytes()
	if err != nil {
		return nil, err
	}
	var exec models.Execution
	json.Unmarshal(data, &exec)
	return &exec, nil
}

func (r *RedisStorage) UpdateExecution(ctx context.Context, namespace string, exec *models.Execution) error {
	return r.CreateExecution(ctx, namespace, exec)
}

func (r *RedisStorage) ListExecutions(ctx context.Context, namespace string) ([]*models.Execution, error) {
	// Simple MVP implementation: search for keys matching the namespace execution pattern
	pattern := fmt.Sprintf("ex:%s:*", namespace)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var execs []*models.Execution
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var exec models.Execution
		json.Unmarshal(data, &exec)
		execs = append(execs, &exec)
	}
	return execs, nil
}

func (r *RedisStorage) CreateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error {
	step.UpdatedAt = time.Now()
	data, _ := json.Marshal(step)
	err := r.client.Set(ctx, stepKey(namespace, step.ID), data, 0).Err()
	if err != nil {
		return err
	}

	// Add to pending sorted set for workers
	score := float64(time.Now().Unix())
	if step.ScheduledAt != nil {
		score = float64(step.ScheduledAt.Unix())
	}
	return r.client.ZAdd(ctx, pendingSet(namespace), redis.Z{
		Score:  score,
		Member: step.ID,
	}).Err()
}

func (r *RedisStorage) UpdateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error {
	step.UpdatedAt = time.Now()
	data, _ := json.Marshal(step)
	err := r.client.Set(ctx, stepKey(namespace, step.ID), data, 0).Err()
	if err != nil {
		return err
	}

	// If finished, remove from pending
	if step.Status != models.TaskPending {
		r.client.ZRem(ctx, pendingSet(namespace), step.ID)
	}
	return nil
}

func (r *RedisStorage) GetExecutionSteps(ctx context.Context, namespace string, executionID string) ([]*models.ExecutionStep, error) {
	// In production we would maintain a set of step IDs per execution
	// For MVP/Verification, we use KEYS (expensive but okay for small scale)
	pattern := fmt.Sprintf("step:%s:*", namespace)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var steps []*models.ExecutionStep
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var step models.ExecutionStep
		json.Unmarshal(data, &step)
		if step.ExecutionID == executionID {
			steps = append(steps, &step)
		}
	}
	return steps, nil
}

func (r *RedisStorage) ClaimReadyStep(ctx context.Context, namespace string, workerID string) (*models.ExecutionStep, error) {
	// Atomic Claim: ZREVRANGEBYSCORE + ZREM (or LUA script for safety)
	// For MVP, we fetch one ready item
	now := float64(time.Now().Unix())
	res, err := r.client.ZRangeByScore(ctx, pendingSet(namespace), &redis.ZRangeBy{
		Min:    "-inf",
		Max:    fmt.Sprintf("%f", now),
		Offset: 0,
		Count:  1,
	}).Result()

	if err != nil || len(res) == 0 {
		return nil, nil
	}

	stepID := res[0]
	// Try to remove it to 'claim' it (Distributed lock simulation)
	n, err := r.client.ZRem(ctx, pendingSet(namespace), stepID).Result()
	if err != nil || n == 0 {
		return nil, nil // Already claimed by another worker
	}

	step, err := r.GetExecutionStepByID(ctx, namespace, stepID)
	if err != nil {
		return nil, err
	}

	step.Status = models.TaskRunning
	step.WorkerID = workerID
	step.StartedAt = time.Now()
	r.UpdateExecutionStep(ctx, namespace, step)

	return step, nil
}

func (r *RedisStorage) GetExecutionStepByID(ctx context.Context, namespace string, id string) (*models.ExecutionStep, error) {
	data, err := r.client.Get(ctx, stepKey(namespace, id)).Bytes()
	if err != nil {
		return nil, err
	}
	var step models.ExecutionStep
	json.Unmarshal(data, &step)
	return &step, nil
}
