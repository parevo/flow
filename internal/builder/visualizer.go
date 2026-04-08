package builder

import (
	"fmt"
	"strings"

	"github.com/parevo/flow/internal/models"
)

// ToMermaid generates a Mermaid.js flowchart representation of the workflow.
func ToMermaid(wf *models.Workflow) string {
	var sb strings.Builder

	sb.WriteString("flowchart TD\n")

	// 1. Add Nodes
	for _, n := range wf.Nodes {
		label := n.Name
		if label == "" {
			label = n.ID
		}

		shape := "[%s]"
		switch n.Type {
		case "condition":
			shape = "{" + "{%s}" + "}" // { {Label}} is a diamond in Mermaid
		case "signal":
			shape = "([%s])" // Oval for signals
		case "subworkflow":
			shape = "[/%s/]" // Parallelogram for sub-workflows
		}

		sb.WriteString(fmt.Sprintf("    %s"+shape+"\n", n.ID, label))
	}

	// 2. Add Edges
	for _, e := range wf.Edges {
		if e.Condition != "" {
			sb.WriteString(fmt.Sprintf("    %s -- %s --> %s\n", e.SourceID, e.Condition, e.TargetID))
		} else {
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", e.SourceID, e.TargetID))
		}
	}

	return sb.String()
}
