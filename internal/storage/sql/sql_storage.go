package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/parevo/flow/internal/models"
	"github.com/parevo/flow/internal/storage"
)

type Dialect interface {
	Placeholder(n int) string
	SkipLocked(query string) string
}

type MySQLDialect struct{}

func (d MySQLDialect) Placeholder(n int) string { return "?" }
func (d MySQLDialect) SkipLocked(query string) string {
	return query + " FOR UPDATE SKIP LOCKED"
}

type PostgresDialect struct{}

func (d PostgresDialect) Placeholder(n int) string { return fmt.Sprintf("$%d", n) }
func (d PostgresDialect) SkipLocked(query string) string {
	return query + " FOR UPDATE SKIP LOCKED"
}

type SQLStorage struct {
	db      *sqlx.DB
	dialect Dialect
	crypto  *storage.Crypto // Optional encryption
}

func NewSQLStorage(db *sqlx.DB, dialectName string) (*SQLStorage, error) {
	var dialect Dialect
	switch dialectName {
	case "mysql":
		dialect = MySQLDialect{}
	case "postgres":
		dialect = PostgresDialect{}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialectName)
	}
	return &SQLStorage{db: db, dialect: dialect}, nil
}

func (s *SQLStorage) SetEncryption(crypto *storage.Crypto) {
	s.crypto = crypto
}

func (s *SQLStorage) SaveWorkflow(ctx context.Context, namespace string, wf *models.Workflow) error {
	wf.Namespace = namespace
	definitionJSON, err := json.Marshal(wf)
	if err != nil {
		return err
	}

	query := `INSERT INTO workflows (id, namespace, name, description, version, status, definition, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = `INSERT INTO workflows (id, namespace, name, description, version, status, definition, created_at, updated_at)
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	}

	_, err = s.db.ExecContext(ctx, query,
		wf.ID, wf.Namespace, wf.Name, wf.Description, wf.Version, wf.Status, definitionJSON, wf.CreatedAt, wf.UpdatedAt)
	return err
}

func (s *SQLStorage) GetWorkflow(ctx context.Context, namespace string, id string) (*models.Workflow, error) {
	query := "SELECT definition FROM workflows WHERE id = ? AND namespace = ?"
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = "SELECT definition FROM workflows WHERE id = $1 AND namespace = $2"
	}

	var definitionJSON []byte
	err := s.db.GetContext(ctx, &definitionJSON, query, id, namespace)
	if err != nil {
		return nil, err
	}

	var wf models.Workflow
	if err := json.Unmarshal(definitionJSON, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

func (s *SQLStorage) ListWorkflows(ctx context.Context, namespace string) ([]*models.Workflow, error) {
	query := "SELECT definition FROM workflows WHERE namespace = ?"
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = "SELECT definition FROM workflows WHERE namespace = $1"
	}
	var definitions [][]byte
	err := s.db.SelectContext(ctx, &definitions, query, namespace)
	if err != nil {
		return nil, err
	}

	var wfs []*models.Workflow
	for _, d := range definitions {
		var wf models.Workflow
		if err := json.Unmarshal(d, &wf); err != nil {
			return nil, err
		}
		wfs = append(wfs, &wf)
	}
	return wfs, nil
}

func (s *SQLStorage) CreateExecution(ctx context.Context, namespace string, exec *models.Execution) error {
	exec.Namespace = namespace
	input := exec.Input
	if s.crypto != nil {
		encryptedInput, err := s.crypto.Encrypt(input)
		if err != nil {
			return err
		}
		input = encryptedInput
	}

	query := `INSERT INTO executions (id, namespace, workflow_id, version, status, input, started_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = `INSERT INTO executions (id, namespace, workflow_id, version, status, input, started_at)
				  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	}

	_, err := s.db.ExecContext(ctx, query,
		exec.ID, exec.Namespace, exec.WorkflowID, exec.Version, exec.Status, input, exec.StartedAt)
	return err
}

func (s *SQLStorage) GetExecution(ctx context.Context, namespace string, id string) (*models.Execution, error) {
	query := "SELECT * FROM executions WHERE id = ? AND namespace = ?"
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = "SELECT * FROM executions WHERE id = $1 AND namespace = $2"
	}
	var exec models.Execution
	err := s.db.GetContext(ctx, &exec, query, id, namespace)
	if err != nil {
		return nil, err
	}

	if s.crypto != nil {
		decryptedInput, _ := s.crypto.Decrypt(exec.Input)
		exec.Input = decryptedInput
		decryptedOutput, _ := s.crypto.Decrypt(exec.Output)
		exec.Output = decryptedOutput
	}

	return &exec, nil
}

func (s *SQLStorage) UpdateExecution(ctx context.Context, namespace string, exec *models.Execution) error {
	output := exec.Output
	if s.crypto != nil {
		encryptedOutput, _ := s.crypto.Encrypt(output)
		output = encryptedOutput
	}

	query := `UPDATE executions SET status = ?, output = ?, error_message = ?, finished_at = ? WHERE id = ? AND namespace = ?`
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = `UPDATE executions SET status = $1, output = $2, error_message = $3, finished_at = $4 WHERE id = $5 AND namespace = $6`
	}

	_, err := s.db.ExecContext(ctx, query,
		exec.Status, output, exec.ErrorMessage, exec.FinishedAt, exec.ID, namespace)
	return err
}

func (s *SQLStorage) ListExecutions(ctx context.Context, namespace string) ([]*models.Execution, error) {
	query := "SELECT * FROM executions WHERE namespace = ?"
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = "SELECT * FROM executions WHERE namespace = $1"
	}
	var execs []*models.Execution
	err := s.db.SelectContext(ctx, &execs, query, namespace)
	if err != nil {
		return nil, err
	}

	if s.crypto != nil {
		for _, exec := range execs {
			dInput, _ := s.crypto.Decrypt(exec.Input)
			exec.Input = dInput
			dOutput, _ := s.crypto.Decrypt(exec.Output)
			exec.Output = dOutput
		}
	}

	return execs, nil
}

func (s *SQLStorage) CreateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error {
	step.Namespace = namespace
	input := step.Input
	if s.crypto != nil {
		encryptedInput, _ := s.crypto.Encrypt(input)
		input = encryptedInput
	}

	query := `INSERT INTO execution_steps (id, namespace, execution_id, node_id, status, input, scheduled_at, started_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = `INSERT INTO execution_steps (id, namespace, execution_id, node_id, status, input, scheduled_at, started_at, updated_at)
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	}
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		step.ID, step.Namespace, step.ExecutionID, step.NodeID, step.Status, input, step.ScheduledAt, step.StartedAt, now)
	return err
}

func (s *SQLStorage) UpdateExecutionStep(ctx context.Context, namespace string, step *models.ExecutionStep) error {
	output := step.Output
	errStr := step.Error
	if s.crypto != nil {
		eOutput, _ := s.crypto.Encrypt(output)
		output = eOutput
		eError, _ := s.crypto.Encrypt(errStr)
		errStr = eError
	}

	query := `UPDATE execution_steps SET status = ?, output = ?, error = ?, finished_at = ?, attempt_number = ?, scheduled_at = ?, updated_at = ? WHERE id = ? AND namespace = ?`
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = `UPDATE execution_steps SET status = $1, output = $2, error = $3, finished_at = $4, attempt_number = $5, scheduled_at = $6, updated_at = $7 WHERE id = $8 AND namespace = $9`
	}
	_, err := s.db.ExecContext(ctx, query,
		step.Status, output, errStr, step.FinishedAt, step.AttemptNumber, step.ScheduledAt, time.Now(), step.ID, namespace)
	return err
}

func (s *SQLStorage) GetExecutionSteps(ctx context.Context, namespace string, executionID string) ([]*models.ExecutionStep, error) {
	query := "SELECT * FROM execution_steps WHERE execution_id = ? AND namespace = ?"
	if _, ok := s.dialect.(PostgresDialect); ok {
		query = "SELECT * FROM execution_steps WHERE execution_id = $1 AND namespace = $2"
	}
	var steps []*models.ExecutionStep
	err := s.db.SelectContext(ctx, &steps, query, executionID, namespace)
	if err != nil {
		return nil, err
	}

	if s.crypto != nil {
		for _, step := range steps {
			dInput, _ := s.crypto.Decrypt(step.Input)
			step.Input = dInput
			dOutput, _ := s.crypto.Decrypt(step.Output)
			step.Output = dOutput
			dError, _ := s.crypto.Decrypt(step.Error)
			step.Error = dError
		}
	}

	return steps, err
}

func (s *SQLStorage) ClaimReadyStep(ctx context.Context, namespace string, workerID string) (*models.ExecutionStep, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Zombie Task Recovery Logic:
	// A task is claimable if:
	// 1. It's PENDING and scheduled_at is now or in the past.
	// 2. It's RUNNING but hasn't been updated for more than 5 minutes (Zombie/Worker Dead).
	
	// MySQL Version
	query := "SELECT * FROM execution_steps WHERE (status = 'PENDING' AND (scheduled_at IS NULL OR scheduled_at <= ?)) OR (status = 'RUNNING' AND updated_at <= ?)"
	args := []interface{}{time.Now(), time.Now().Add(-5 * time.Minute)}
	
	if namespace != "" {
		query = "SELECT * FROM execution_steps WHERE ((status = 'PENDING' AND (scheduled_at IS NULL OR scheduled_at <= ?)) OR (status = 'RUNNING' AND updated_at <= ?)) AND namespace = ?"
		args = append(args, namespace)
	}

	// Postgres Version
	if _, ok := s.dialect.(PostgresDialect); ok {
		if namespace != "" {
			query = "SELECT * FROM execution_steps WHERE ((status = 'PENDING' AND (scheduled_at IS NULL OR scheduled_at <= $1)) OR (status = 'RUNNING' AND updated_at <= $2)) AND namespace = $3"
		} else {
			query = "SELECT * FROM execution_steps WHERE (status = 'PENDING' AND (scheduled_at IS NULL OR scheduled_at <= $1)) OR (status = 'RUNNING' AND updated_at <= $2)"
		}
	}

	query += " LIMIT 1"
	query = s.dialect.SkipLocked(query)

	var step models.ExecutionStep
	err = tx.GetContext(ctx, &step, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	updateQuery := "UPDATE execution_steps SET status = ?, worker_id = ?, started_at = ? WHERE id = ? AND namespace = ?"
	if _, ok := s.dialect.(PostgresDialect); ok {
		updateQuery = "UPDATE execution_steps SET status = $1, worker_id = $2, started_at = $3 WHERE id = $4 AND namespace = $5"
	}

	_, err = tx.ExecContext(ctx, updateQuery, "RUNNING", workerID, time.Now(), step.ID, step.Namespace)
	if err != nil {
		return nil, err
	}

	step.Status = models.TaskRunning
	step.WorkerID = workerID
	
	return &step, tx.Commit()
}
