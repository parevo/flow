-- schema.postgres.sql

CREATE TABLE workflows (
    id UUID PRIMARY KEY,
    namespace VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    definition JSONB NOT NULL,
    labels JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_workflows_namespace (namespace)
);

CREATE TABLE executions (
    id UUID PRIMARY KEY,
    namespace VARCHAR(255) NOT NULL,
    workflow_id UUID NOT NULL,
    version INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    input JSONB,
    output JSONB,
    labels JSONB,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE,
    INDEX idx_executions_namespace_status (namespace, status),
    INDEX idx_executions_workflow (workflow_id)
);

CREATE TABLE execution_steps (
    id UUID PRIMARY KEY,
    namespace VARCHAR(255) NOT NULL,
    execution_id UUID NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    input JSONB,
    output JSONB,
    labels JSONB,
    error TEXT,
    attempt_number INT DEFAULT 0,
    worker_id VARCHAR(255),
    scheduled_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE,
    INDEX idx_steps_namespace_status_exec (namespace, status, execution_id),
    INDEX idx_steps_worker (worker_id)
);
