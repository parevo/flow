package engine

import (
	"encoding/json"
	"fmt"

	"github.com/parevo/flow/internal/models"
)

// Graph represents the structure of the workflow
type Graph struct {
	Nodes        map[string]models.Node
	Edges        map[string][]models.Edge // source -> outgoing edges
	InDegree     map[string]int           // target -> number of incoming edges
	Predecessors map[string][]string      // target -> list of parent node IDs
}

// NewGraph builds a graph from workflow definition
func NewGraph(wf *models.Workflow) (*Graph, error) {
	g := &Graph{
		Nodes:        make(map[string]models.Node),
		Edges:        make(map[string][]models.Edge),
		InDegree:     make(map[string]int),
		Predecessors: make(map[string][]string),
	}

	for _, node := range wf.Nodes {
		g.Nodes[node.ID] = node
		g.InDegree[node.ID] = 0
		g.Predecessors[node.ID] = []string{}
	}

	for _, edge := range wf.Edges {
		if _, ok := g.Nodes[edge.SourceID]; !ok {
			return nil, fmt.Errorf("invalid source id: %s", edge.SourceID)
		}
		if _, ok := g.Nodes[edge.TargetID]; !ok {
			return nil, fmt.Errorf("invalid target id: %s", edge.TargetID)
		}
		g.Edges[edge.SourceID] = append(g.Edges[edge.SourceID], edge)
		g.InDegree[edge.TargetID]++
		g.Predecessors[edge.TargetID] = append(g.Predecessors[edge.TargetID], edge.SourceID)
	}

	return g, nil
}

// Validate checks for cycles in the graph
func (g *Graph) Validate() error {
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

		for _, edge := range g.Edges[curr] {
			target := edge.TargetID
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

	// Find all nodes that are referenced as a CompensateNodeID
	compensations := make(map[string]bool)
	for _, node := range g.Nodes {
		if node.CompensateNodeID != "" {
			compensations[node.CompensateNodeID] = true
		}
	}

	for id, d := range g.InDegree {
		if d == 0 && !compensations[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetNextNodes returns direct children of the given node (standard routing)
func (g *Graph) GetNextNodes(nodeID string) []string {
	var ids []string
	for _, edge := range g.Edges[nodeID] {
		ids = append(ids, edge.TargetID)
	}
	return ids
}

// GetNextNodesWithBranch returns children based on output branch (intelligent routing)
func (g *Graph) GetNextNodesWithBranch(nodeID string, output string) []string {
	var ids []string

	// Parse output to see if it specifies a branch
	var outMap map[string]interface{}
	_ = json.Unmarshal([]byte(output), &outMap)

	branch, hasBranch := outMap["branch"].(string)

	for _, edge := range g.Edges[nodeID] {
		// If node specified a branch, only follow matching edges
		if hasBranch {
			if edge.Condition == branch {
				ids = append(ids, edge.TargetID)
			}
		} else {
			// No branch specified, follow all unlabeled edges
			if edge.Condition == "" {
				ids = append(ids, edge.TargetID)
			}
		}
	}
	return ids
}
