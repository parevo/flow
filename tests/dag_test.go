package tests

import (
	"testing"
	"github.com/parevo/flow/internal/engine"
	"github.com/parevo/flow/internal/models"
)

func TestGraph_Validate(t *testing.T) {
	tests := []struct {
		name    string
		wf      *models.Workflow
		wantErr bool
	}{
		{
			name: "Simple DAG",
			wf: &models.Workflow{
				Nodes: []models.Node{{ID: "1"}, {ID: "2"}},
				Edges: []models.Edge{{ID: "e1", SourceID: "1", TargetID: "2"}},
			},
			wantErr: false,
		},
		{
			name: "Cycle Directed",
			wf: &models.Workflow{
				Nodes: []models.Node{{ID: "1"}, {ID: "2"}},
				Edges: []models.Edge{{ID: "e1", SourceID: "1", TargetID: "2"}, {ID: "e2", SourceID: "2", TargetID: "1"}},
			},
			wantErr: true,
		},
		{
			name: "Diamond Shape",
			wf: &models.Workflow{
				Nodes: []models.Node{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}},
				Edges: []models.Edge{
					{ID: "e1", SourceID: "1", TargetID: "2"},
					{ID: "e2", SourceID: "1", TargetID: "3"},
					{ID: "e3", SourceID: "2", TargetID: "4"},
					{ID: "e4", SourceID: "3", TargetID: "4"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := engine.NewGraph(tt.wf)
			if err != nil {
				t.Fatalf("Failed to create graph: %v", err)
			}
			if err := g.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Graph.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGraph_GetInitialNodes(t *testing.T) {
	wf := &models.Workflow{
		Nodes: []models.Node{{ID: "1"}, {ID: "2"}, {ID: "3"}},
		Edges: []models.Edge{{ID: "e1", SourceID: "1", TargetID: "2"}},
	}
	g, _ := engine.NewGraph(wf)
	initial := g.GetInitialNodes()
	if len(initial) != 2 { // Node 1 and Node 3 are initial
		t.Errorf("Expected 2 initial nodes, got %d", len(initial))
	}
}
