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
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflows_namespace ON workflows(namespace);

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
    finished_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_executions_namespace_status ON executions(namespace, status);
CREATE INDEX idx_executions_workflow ON executions(workflow_id);

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
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_steps_namespace_status_sch ON execution_steps(namespace, status, scheduled_at);
CREATE INDEX idx_steps_status_updated ON execution_steps(status, updated_at);
CREATE INDEX idx_steps_execution_id ON execution_steps(execution_id);
CREATE INDEX idx_steps_worker ON execution_steps(worker_id);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_execution_steps_updated_at BEFORE UPDATE ON execution_steps FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
