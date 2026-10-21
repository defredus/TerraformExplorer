package hclparser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const maxTerraformFileSize int64 = 10 << 20

// ScanDir parses the native-syntax .tf files directly contained in dir. It
// never evaluates expressions, follows module sources, or executes Terraform.
func ScanDir(dir string) (Result, error) {
	root, err := canonicalDirectory(dir)
	if err != nil {
		return Result{}, err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		SchemaVersion: SchemaVersion,
		Module:        Module{Address: RootModuleAddress, Path: root},
		Modules:       []Module{{Address: RootModuleAddress, Path: root}},
		Files:         make([]File, 0),
		Declarations:  make([]Declaration, 0),
		Diagnostics:   make([]Diagnostic, 0),
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		path := filepath.Join(root, entry.Name())
		if entry.Type()&os.ModeSymlink != 0 {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "warning",
				Summary:  "Symbolic link skipped",
				Detail:   fmt.Sprintf("Terraform file %q is a symbolic link; links are not followed in read-only scan mode.", entry.Name()),
			})
			continue
		}

		info, err := entry.Info()
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, fileDiagnostic(entry.Name(), "Unable to inspect file", err))
			continue
		}
		if !info.Mode().IsRegular() {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "warning",
				Summary:  "Non-regular file skipped",
				Detail:   fmt.Sprintf("Terraform path %q is not a regular file.", entry.Name()),
			})
			continue
		}
		if info.Size() > maxTerraformFileSize {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Summary:  "Terraform file is too large",
				Detail:   fmt.Sprintf("File %q is %d bytes; the scan limit is %d bytes.", entry.Name(), info.Size(), maxTerraformFileSize),
			})
			continue
		}

		source, err := os.ReadFile(path)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, fileDiagnostic(entry.Name(), "Unable to read file", err))
			continue
		}

		result.Files = append(result.Files, File{
			Path:          entry.Name(),
			Bytes:         int64(len(source)),
			ModuleAddress: RootModuleAddress,
			SourcePath:    path,
		})
		parseFile(entry.Name(), path, source, &result)
	}

	sortResult(&result)
	return result, nil
}

func canonicalDirectory(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", &os.PathError{Op: "scan", Path: dir, Err: fmt.Errorf("not a directory")}
	}
	return filepath.Clean(real), nil
}

func parseFile(relativePath, absolutePath string, source []byte, result *Result) {
	parser := hclparse.NewParser()
	file, diagnostics := parser.ParseHCL(source, absolutePath)
	appendHCLDiagnostics(relativePath, diagnostics, result)
	if file == nil || file.Body == nil {
		return
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Summary:  "Unsupported HCL body",
			Detail:   fmt.Sprintf("File %q did not produce a native HCL syntax tree.", relativePath),
		})
		return
	}

	for _, block := range body.Blocks {
		extractBlock(relativePath, source, block, result)
	}
}

func extractBlock(file string, source []byte, block *hclsyntax.Block, result *Result) {
	switch block.Type {
	case "resource":
		if !requireLabels(file, block, 2, result) {
			return
		}
		appendBlockDeclaration(result, DeclarationResource, block.Labels[1], block.Labels[0], block, file, source, block.LabelRanges[1])
	case "data":
		if !requireLabels(file, block, 2, result) {
			return
		}
		appendBlockDeclaration(result, DeclarationData, block.Labels[1], block.Labels[0], block, file, source, block.LabelRanges[1])
	case "module":
		if !requireLabels(file, block, 1, result) {
			return
		}
		appendBlockDeclaration(result, DeclarationModule, block.Labels[0], "", block, file, source, block.LabelRanges[0])
		result.Declarations[len(result.Declarations)-1].ModuleSource = staticModuleSource(block)
	case "variable":
		if !requireLabels(file, block, 1, result) {
			return
		}
		appendBlockDeclaration(result, DeclarationVariable, block.Labels[0], "", block, file, source, block.LabelRanges[0])
	case "output":
		if !requireLabels(file, block, 1, result) {
			return
		}
		appendBlockDeclaration(result, DeclarationOutput, block.Labels[0], "", block, file, source, block.LabelRanges[0])
	case "locals":
		for _, attribute := range block.Body.Attributes {
			result.Declarations = append(result.Declarations, Declaration{
				Kind:           DeclarationLocal,
				Address:        "local." + attribute.Name,
				ModuleAddress:  RootModuleAddress,
				Name:           attribute.Name,
				File:           file,
				Range:          sourceRange(file, attribute.Range()),
				SelectionRange: sourceRange(file, attribute.NameRange),
				References:     expressionReferences(file, source, attribute.Name, attribute.Expr),
			})
		}
	}
}

func requireLabels(file string, block *hclsyntax.Block, want int, result *Result) bool {
	if len(block.Labels) == want {
		return true
	}

	rng := sourceRange(file, block.TypeRange)
	result.Diagnostics = append(result.Diagnostics, Diagnostic{
		Severity: "error",
		Summary:  "Invalid Terraform block labels",
		Detail:   fmt.Sprintf("Block %q has %d labels; this scanner expects %d.", block.Type, len(block.Labels), want),
		Subject:  &rng,
	})
	return false
}

func appendBlockDeclaration(result *Result, kind DeclarationKind, name, typeName string, block *hclsyntax.Block, file string, source []byte, selection hcl.Range) {
	result.Declarations = append(result.Declarations, Declaration{
		Kind:           kind,
		Address:        declarationAddress(kind, typeName, name),
		ModuleAddress:  RootModuleAddress,
		Name:           name,
		Type:           typeName,
		File:           file,
		Range:          sourceRange(file, block.Range()),
		SelectionRange: sourceRange(file, selection),
		References:     bodyReferences(file, source, block.Body, ""),
	})
}

func staticModuleSource(block *hclsyntax.Block) string {
	attribute, ok := block.Body.Attributes["source"]
	if !ok {
		return ""
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || !value.IsKnown() || value.IsNull() || value.Type() != cty.String {
		return ""
	}
	return value.AsString()
}

func bodyReferences(file string, source []byte, body *hclsyntax.Body, prefix string) []Reference {
	references := make([]Reference, 0)
	for name, attribute := range body.Attributes {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		references = append(references, expressionReferences(file, source, path, attribute.Expr)...)
	}
	for _, block := range body.Blocks {
		path := block.Type
		if prefix != "" {
			path = prefix + "." + block.Type
		}
		references = append(references, bodyReferences(file, source, block.Body, path)...)
	}
	sort.SliceStable(references, func(i, j int) bool {
		return references[i].Range.Start.Byte < references[j].Range.Start.Byte
	})
	return references
}

func expressionReferences(file string, source []byte, attributePath string, expression hclsyntax.Expression) []Reference {
	variables := expression.Variables()
	references := make([]Reference, 0, len(variables))
	for _, traversal := range variables {
		rng := traversal.SourceRange()
		start, end := rng.Start.Byte, rng.End.Byte
		text := ""
		if start >= 0 && end >= start && end <= len(source) {
			text = string(source[start:end])
		}

		parts := make([]string, 0, len(traversal))
		for _, traverser := range traversal {
			switch step := traverser.(type) {
			case hcl.TraverseRoot:
				parts = append(parts, step.Name)
			case hcl.TraverseAttr:
				parts = append(parts, step.Name)
			}
		}

		references = append(references, Reference{
			Text:          text,
			Traversal:     parts,
			AttributePath: attributePath,
			Range:         sourceRange(file, rng),
		})
	}
	return references
}

func declarationAddress(kind DeclarationKind, typeName, name string) string {
	switch kind {
	case DeclarationResource:
		return typeName + "." + name
	case DeclarationData:
		return "data." + typeName + "." + name
	case DeclarationModule:
		return "module." + name
	case DeclarationVariable:
		return "var." + name
	case DeclarationOutput:
		return "output." + name
	default:
		return name
	}
}

func appendHCLDiagnostics(file string, diagnostics hcl.Diagnostics, result *Result) {
	for _, diagnostic := range diagnostics {
		item := Diagnostic{
			Severity: severityName(diagnostic.Severity),
			Summary:  diagnostic.Summary,
			Detail:   diagnostic.Detail,
		}
		if diagnostic.Subject != nil {
			rng := sourceRange(file, *diagnostic.Subject)
			item.Subject = &rng
		}
		result.Diagnostics = append(result.Diagnostics, item)
	}
}

func severityName(severity hcl.DiagnosticSeverity) string {
	if severity == hcl.DiagError {
		return "error"
	}
	return "warning"
}

func sourceRange(file string, value hcl.Range) Range {
	return Range{
		File:  file,
		Start: Position{Line: value.Start.Line, Column: value.Start.Column, Byte: value.Start.Byte},
		End:   Position{Line: value.End.Line, Column: value.End.Column, Byte: value.End.Byte},
	}
}

func fileDiagnostic(file, summary string, err error) Diagnostic {
	return Diagnostic{
		Severity: "error",
		Summary:  summary,
		Detail:   fmt.Sprintf("%s: %v", file, err),
	}
}

func sortResult(result *Result) {
	sort.Slice(result.Files, func(i, j int) bool {
		return result.Files[i].Path < result.Files[j].Path
	})
	sort.SliceStable(result.Declarations, func(i, j int) bool {
		left, right := result.Declarations[i], result.Declarations[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Range.Start.Byte != right.Range.Start.Byte {
			return left.Range.Start.Byte < right.Range.Start.Byte
		}
		return left.Address < right.Address
	})
	for index := range result.Declarations {
		declaration := &result.Declarations[index]
		if declaration.References == nil {
			declaration.References = make([]Reference, 0)
		}
	}
}
