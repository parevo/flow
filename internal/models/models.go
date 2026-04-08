package models

import (
	"time"
)

type WorkflowStatus string

const (
	WorkflowActive   WorkflowStatus = "ACTIVE"
	WorkflowInactive WorkflowStatus = "INACTIVE"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "PENDING"
	TaskRunning   TaskStatus = "RUNNING"
	TaskCompleted TaskStatus = "COMPLETED"
	TaskFailed    TaskStatus = "FAILED"
	TaskSkipped   TaskStatus = "SKIPPED"
	TaskWaiting   TaskStatus = "WAITING"
)

// Workflow represents the blueprint of a workflow
type Workflow struct {
	ID          string            `json:"id" db:"id"`
	Namespace   string            `json:"namespace" db:"namespace"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Version     int               `json:"version" db:"version"`
	Status      WorkflowStatus    `json:"status" db:"status"`
	Nodes       []Node            `json:"nodes" db:"-"` // Handled separately or as JSON
	Edges       []Edge            `json:"edges" db:"-"` // Handled separately or as JSON
	Labels      map[string]string `json:"labels" db:"-"` // Stored in JSON definition
	CreatedAt   time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time         `json:"updatedAt" db:"updated_at"`
}

// Node represents a single step in a workflow
type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // e.g., "http", "script", "delay"
	Name       string                 `json:"name"`
	Config     map[string]interface{} `json:"config"`
	RetryCount int                    `json:"retryCount"`
}

// Edge represents a connection between two nodes
type Edge struct {
	ID         string `json:"id"`
	SourceID   string `json:"sourceId"`
	TargetID   string `json:"targetId"`
	Condition  string `json:"condition,omitempty"` // e.g., "success", "failure", or a custom expression
}

// Execution represents a single run instance of a workflow
type Execution struct {
	ID           string            `json:"id" db:"id"`
	Namespace    string            `json:"namespace" db:"namespace"`
	WorkflowID   string            `json:"workflowId" db:"workflow_id"`
	Version      int               `json:"version" db:"version"`
	Status       TaskStatus        `json:"status" db:"status"`
	Input        string            `json:"input" db:"input"`
	Output       string            `json:"output" db:"output"`
	ErrorMessage string            `json:"errorMessage,omitempty" db:"error_message"`
	Labels       map[string]string `json:"labels" db:"-"` // Stored in JSON or separate column
	StartedAt    time.Time         `json:"startedAt" db:"started_at"`
	FinishedAt   *time.Time        `json:"finishedAt,omitempty" db:"finished_at"`
}

// ExecutionStep represents the state of an individual node in an execution
type ExecutionStep struct {
	ID            string            `json:"id" db:"id"`
	Namespace     string            `json:"namespace" db:"namespace"`
	ExecutionID   string            `json:"executionId" db:"execution_id"`
	NodeID        string            `json:"nodeId" db:"node_id"`
	Status        TaskStatus        `json:"status" db:"status"`
	Input         string            `json:"input" db:"input"`
	Output        string            `json:"output" db:"output"`
	Error         string            `json:"error,omitempty" db:"error"`
	AttemptNumber int               `json:"attemptNumber" db:"attempt_number"`
	WorkerID      string            `json:"workerId,omitempty" db:"worker_id"`
	Labels        map[string]string `json:"labels" db:"-"`
	StartedAt     time.Time         `json:"startedAt" db:"started_at"`
	FinishedAt    *time.Time        `json:"finishedAt,omitempty" db:"finished_at"`
}
