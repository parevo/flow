package builder

import (
	"github.com/google/uuid"
	"github.com/parevo/flow/internal/models"
)

type WorkflowBuilder struct {
	workflow *models.Workflow
}

type NodeBuilder struct {
	builder *WorkflowBuilder
	node    *models.Node
}

func NewWorkflow(id, name string) *WorkflowBuilder {
	return &WorkflowBuilder{
		workflow: &models.Workflow{
			ID:     id,
			Name:   name,
			Nodes:  []models.Node{},
			Edges:  []models.Edge{},
			Status: models.WorkflowActive,
		},
	}
}

func (b *WorkflowBuilder) AddNode(id, nodeType string) *NodeBuilder {
	n := models.Node{
		ID:     id,
		Type:   nodeType,
		Config: make(map[string]interface{}),
	}
	b.workflow.Nodes = append(b.workflow.Nodes, n)
	return &NodeBuilder{
		builder: b,
		node:    &b.workflow.Nodes[len(b.workflow.Nodes)-1],
	}
}

func (nb *NodeBuilder) WithConfig(key string, value interface{}) *NodeBuilder {
	nb.node.Config[key] = value
	return nb
}

func (nb *NodeBuilder) WithSaga(compensateNodeID string) *NodeBuilder {
	nb.node.CompensateNodeID = compensateNodeID
	return nb
}

func (nb *NodeBuilder) WithRetry(count int) *NodeBuilder {
	nb.node.RetryCount = count
	return nb
}

func (nb *NodeBuilder) Then(targetID string) *NodeBuilder {
	b := nb.builder
	edge := models.Edge{
		ID:       uuid.New().String(),
		SourceID: nb.node.ID,
		TargetID: targetID,
	}
	b.workflow.Edges = append(b.workflow.Edges, edge)
	
	// Find and return the target node builder if it exists
	for i := range b.workflow.Nodes {
		if b.workflow.Nodes[i].ID == targetID {
			return &NodeBuilder{builder: b, node: &b.workflow.Nodes[i]}
		}
	}
	return nb
}

func (nb *NodeBuilder) If(targetID string, condition string) *NodeBuilder {
	b := nb.builder
	edge := models.Edge{
		ID:        uuid.New().String(),
		SourceID:  nb.node.ID,
		TargetID:  targetID,
		Condition: condition,
	}
	b.workflow.Edges = append(b.workflow.Edges, edge)
	
	// Find and return the target node builder
	for i := range b.workflow.Nodes {
		if b.workflow.Nodes[i].ID == targetID {
			return &NodeBuilder{builder: b, node: &b.workflow.Nodes[i]}
		}
	}
	return nb
}

func (b *WorkflowBuilder) Connect(sourceID, targetID string) *WorkflowBuilder {
	edge := models.Edge{
		ID:       uuid.New().String(),
		SourceID: sourceID,
		TargetID: targetID,
	}
	b.workflow.Edges = append(b.workflow.Edges, edge)
	return b
}

func (b *WorkflowBuilder) ConnectIf(sourceID, targetID string, condition string) *WorkflowBuilder {
	edge := models.Edge{
		ID:        uuid.New().String(),
		SourceID:  sourceID,
		TargetID:  targetID,
		Condition: condition,
	}
	b.workflow.Edges = append(b.workflow.Edges, edge)
	return b
}

func (b *WorkflowBuilder) Build() *models.Workflow {
	return b.workflow
}

func (b *WorkflowBuilder) Visualise() string {
	return ToMermaid(b.workflow)
}
