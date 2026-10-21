package semantic

import (
	"fmt"
	"strings"

	"github.com/terraform-explorer/terraform-explorer/internal/hclparser"
)

type Graph struct {
	Nodes      []Node                `json:"nodes"`
	Edges      []Edge                `json:"edges"`
	Unresolved []UnresolvedReference `json:"unresolved"`
}

type Node struct {
	ID             string                    `json:"id"`
	Address        string                    `json:"address"`
	Label          string                    `json:"label"`
	Kind           hclparser.DeclarationKind `json:"kind"`
	Type           string                    `json:"type,omitempty"`
	File           string                    `json:"file"`
	Range          hclparser.Range           `json:"range"`
	SelectionRange hclparser.Range           `json:"selectionRange"`
}

type Edge struct {
	ID            string          `json:"id"`
	Source        string          `json:"source"`
	Target        string          `json:"target"`
	Kind          string          `json:"kind"`
	Reference     string          `json:"reference"`
	AttributePath string          `json:"attributePath"`
	Range         hclparser.Range `json:"range"`
}

type UnresolvedReference struct {
	ConsumerID string              `json:"consumerId"`
	Address    string              `json:"address,omitempty"`
	Reason     string              `json:"reason"`
	Reference  hclparser.Reference `json:"reference"`
}

// BuildGraph resolves syntax traversals against declarations from one module.
// It intentionally does not evaluate expressions or invent resource instances.
func BuildGraph(result hclparser.Result) Graph {
	graph := Graph{
		Nodes:      make([]Node, 0, len(result.Declarations)),
		Edges:      make([]Edge, 0),
		Unresolved: make([]UnresolvedReference, 0),
	}

	byAddress := make(map[string][]string, len(result.Declarations))
	for _, declaration := range result.Declarations {
		id := nodeID(declaration)
		graph.Nodes = append(graph.Nodes, Node{
			ID:             id,
			Address:        declaration.Address,
			Label:          declaration.Name,
			Kind:           declaration.Kind,
			Type:           declaration.Type,
			File:           declaration.File,
			Range:          declaration.Range,
			SelectionRange: declaration.SelectionRange,
		})
		byAddress[declaration.Address] = append(byAddress[declaration.Address], id)
	}

	for _, declaration := range result.Declarations {
		consumerID := nodeID(declaration)
		for _, reference := range declaration.References {
			address, reason := referenceAddress(reference.Traversal)
			if reason != "" {
				if reason != "type_constraint" {
					graph.Unresolved = append(graph.Unresolved, UnresolvedReference{
						ConsumerID: consumerID,
						Address:    address,
						Reason:     reason,
						Reference:  reference,
					})
				}
				continue
			}

			producers := byAddress[address]
			if len(producers) != 1 {
				reason := "symbol_not_found"
				if len(producers) > 1 {
					reason = "ambiguous_symbol"
				}
				graph.Unresolved = append(graph.Unresolved, UnresolvedReference{
					ConsumerID: consumerID,
					Address:    address,
					Reason:     reason,
					Reference:  reference,
				})
				continue
			}

			graph.Edges = append(graph.Edges, Edge{
				ID:            fmt.Sprintf("edge-%d", len(graph.Edges)+1),
				Source:        producers[0],
				Target:        consumerID,
				Kind:          edgeKind(reference.AttributePath),
				Reference:     reference.Text,
				AttributePath: reference.AttributePath,
				Range:         reference.Range,
			})
		}
	}

	return graph
}

func nodeID(declaration hclparser.Declaration) string {
	return fmt.Sprintf("%s|%s|%s|%d", declaration.Kind, declaration.Address, declaration.File, declaration.Range.Start.Byte)
}

func edgeKind(attributePath string) string {
	if attributePath == "depends_on" || strings.HasSuffix(attributePath, ".depends_on") {
		return "explicit_dependency"
	}
	if attributePath == "count" || attributePath == "for_each" || strings.HasSuffix(attributePath, ".count") || strings.HasSuffix(attributePath, ".for_each") {
		return "cardinality"
	}
	return "value_flow"
}

func referenceAddress(parts []string) (string, string) {
	if len(parts) == 0 {
		return "", "unsupported_traversal"
	}

	switch parts[0] {
	case "var", "local":
		if len(parts) < 2 {
			return strings.Join(parts, "."), "incomplete_reference"
		}
		return strings.Join(parts[:2], "."), ""
	case "data":
		if len(parts) < 3 {
			return strings.Join(parts, "."), "incomplete_reference"
		}
		return strings.Join(parts[:3], "."), ""
	case "module":
		if len(parts) < 2 {
			return strings.Join(parts, "."), "incomplete_reference"
		}
		return strings.Join(parts[:2], "."), ""
	case "terraform", "path", "count", "each", "self":
		return strings.Join(parts, "."), "runtime_symbol"
	case "string", "number", "bool", "any", "list", "set", "map", "object", "tuple", "optional":
		return strings.Join(parts, "."), "type_constraint"
	default:
		if len(parts) < 2 {
			return strings.Join(parts, "."), "unsupported_symbol"
		}
		return strings.Join(parts[:2], "."), ""
	}
}
