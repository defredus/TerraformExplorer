package semantic

import (
	"path/filepath"
	"testing"

	"github.com/terraform-explorer/terraform-explorer/internal/hclparser"
)

func TestBuildGraphResolvesStaticReferences(t *testing.T) {
	result, err := hclparser.ScanDir(filepath.Join("..", "..", "testdata", "simple-module"))
	if err != nil {
		t.Fatal(err)
	}

	graph := BuildGraph(result)
	if got, want := len(graph.Nodes), 7; got != want {
		t.Fatalf("nodes = %d, want %d", got, want)
	}
	if got, want := len(graph.Edges), 4; got != want {
		for _, edge := range graph.Edges {
			t.Logf("edge: %#v", edge)
		}
		t.Fatalf("edges = %d, want %d", got, want)
	}
	if got := len(graph.Unresolved); got != 0 {
		t.Fatalf("unresolved = %d, want 0: %#v", got, graph.Unresolved)
	}

	for _, edge := range graph.Edges {
		if edge.Kind != "value_flow" {
			t.Errorf("edge kind = %q, want value_flow", edge.Kind)
		}
		if edge.Reference == "" || edge.Range.File == "" {
			t.Errorf("edge has incomplete provenance: %#v", edge)
		}
	}
}

func TestBuildGraphClassifiesDependencyKindsAndUnknowns(t *testing.T) {
	producer := hclparser.Declaration{
		Kind:    hclparser.DeclarationResource,
		Address: "aws_vpc.main",
		Name:    "main",
		File:    "main.tf",
		Range:   testRange("main.tf", 0),
	}
	consumer := hclparser.Declaration{
		Kind:    hclparser.DeclarationResource,
		Address: "aws_subnet.main",
		Name:    "main",
		File:    "main.tf",
		Range:   testRange("main.tf", 20),
		References: []hclparser.Reference{
			{Text: "aws_vpc.main", Traversal: []string{"aws_vpc", "main"}, AttributePath: "depends_on", Range: testRange("main.tf", 30)},
			{Text: "var.count", Traversal: []string{"var", "count"}, AttributePath: "count", Range: testRange("main.tf", 40)},
			{Text: "each.key", Traversal: []string{"each", "key"}, AttributePath: "name", Range: testRange("main.tf", 50)},
		},
	}

	graph := BuildGraph(hclparser.Result{Declarations: []hclparser.Declaration{producer, consumer}})
	if got, want := len(graph.Edges), 1; got != want {
		t.Fatalf("edges = %d, want %d: %#v", got, want, graph.Edges)
	}
	if graph.Edges[0].Kind != "explicit_dependency" {
		t.Fatalf("edge kind = %q, want explicit_dependency", graph.Edges[0].Kind)
	}
	if got, want := len(graph.Unresolved), 2; got != want {
		t.Fatalf("unresolved = %d, want %d: %#v", got, want, graph.Unresolved)
	}
	if graph.Unresolved[0].Reason != "symbol_not_found" || graph.Unresolved[1].Reason != "runtime_symbol" {
		t.Fatalf("unexpected unresolved reasons: %#v", graph.Unresolved)
	}
}

func testRange(file string, offset int) hclparser.Range {
	return hclparser.Range{
		File:  file,
		Start: hclparser.Position{Line: 1, Column: offset + 1, Byte: offset},
		End:   hclparser.Position{Line: 1, Column: offset + 2, Byte: offset + 1},
	}
}
