package engine

import (
	"fmt"
	"github.com/parevo/flow/internal/models"
)

// Graph represents the structure of the workflow
type Graph struct {
	Nodes map[string]models.Node
	Edges map[string][]string // source -> targets
	InDegree map[string]int    // target -> number of incoming edges
}

// NewGraph builds a graph from workflow definition
func NewGraph(wf *models.Workflow) (*Graph, error) {
	g := &Graph{
		Nodes:    make(map[string]models.Node),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
	}

	for _, node := range wf.Nodes {
		g.Nodes[node.ID] = node
		g.InDegree[node.ID] = 0
	}

	for _, edge := range wf.Edges {
		if _, ok := g.Nodes[edge.SourceID]; !ok {
			return nil, fmt.Errorf("invalid source id: %s", edge.SourceID)
		}
		if _, ok := g.Nodes[edge.TargetID]; !ok {
			return nil, fmt.Errorf("invalid target id: %s", edge.TargetID)
		}
		g.Edges[edge.SourceID] = append(g.Edges[edge.SourceID], edge.TargetID)
		g.InDegree[edge.TargetID]++
	}

	return g, nil
}

// Validate checks for cycles in the graph
func (g *Graph) Validate() error {
	// Simple Kahn's algorithm for topological sorting
	q := []string{}
	inDegree := make(map[string]int)
	for id, d := range g.InDegree {
		inDegree[id] = d
		if d == 0 {
			q = append(q, id)
		}
	}

	visitedCount := 0
	for len(q) > 0 {
		curr := q[0]
		q = q[1:]
		visitedCount++

		for _, target := range g.Edges[curr] {
			inDegree[target]--
			if inDegree[target] == 0 {
				q = append(q, target)
			}
		}
	}

	if visitedCount != len(g.Nodes) {
		return fmt.Errorf("cycle detected in workflow graph")
	}

	return nil
}

// GetInitialNodes returns nodes with no predecessors
func (g *Graph) GetInitialNodes() []string {
	ids := []string{}
	for id, d := range g.InDegree {
		if d == 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetNextNodes returns direct children of the given node
func (g *Graph) GetNextNodes(nodeID string) []string {
	return g.Edges[nodeID]
}
