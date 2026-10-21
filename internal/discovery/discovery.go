// Package discovery follows local Terraform module calls without executing
// Terraform or downloading external module sources.
package discovery

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/terraform-explorer/terraform-explorer/internal/hclparser"
)

const (
	maxModuleDepth     = 16
	maxModuleInstances = 128
)

// ScanDir scans a root module and recursively instantiates local modules whose
// source starts with ./ or ../ and remains inside the selected root directory.
// Registry, Git, absolute, dynamic, and unavailable sources stay represented
// by their module-call declaration but are never opened.
func ScanDir(dir string) (hclparser.Result, error) {
	root, err := hclparser.ScanDir(dir)
	if err != nil {
		return hclparser.Result{}, err
	}

	result := hclparser.Result{
		SchemaVersion: hclparser.SchemaVersion,
		Module:        root.Module,
		Modules:       make([]hclparser.Module, 0, 4),
		Files:         make([]hclparser.File, 0, len(root.Files)),
		Declarations:  make([]hclparser.Declaration, 0, len(root.Declarations)),
		Diagnostics:   make([]hclparser.Diagnostic, 0, len(root.Diagnostics)),
	}
	state := scanner{rootPath: root.Module.Path, result: &result}
	state.visit(root, hclparser.RootModuleAddress, "", "", 0, map[string]bool{})
	state.sort()
	return result, nil
}

type scanner struct {
	rootPath string
	result   *hclparser.Result
}

func (scanner *scanner) visit(parsed hclparser.Result, address, parent, source string, depth int, ancestors map[string]bool) {
	modulePath := parsed.Module.Path
	module := hclparser.Module{Address: address, Path: modulePath, Parent: parent, Source: source}
	scanner.result.Modules = append(scanner.result.Modules, module)
	if address == hclparser.RootModuleAddress {
		scanner.result.Module = module
	}

	fileNames := make(map[string]string, len(parsed.Files))
	for _, file := range parsed.Files {
		logicalPath := instanceFile(address, file.Path)
		fileNames[file.Path] = logicalPath
		file.Path = logicalPath
		file.ModuleAddress = address
		if file.SourcePath == "" {
			file.SourcePath = filepath.Join(modulePath, filepath.Base(logicalPath))
		}
		scanner.result.Files = append(scanner.result.Files, file)
	}

	declarations := make([]hclparser.Declaration, 0, len(parsed.Declarations))
	for _, declaration := range parsed.Declarations {
		declaration.ModuleAddress = address
		declaration.File = logicalFile(fileNames, declaration.File, address)
		declaration.Range = rebaseRange(declaration.Range, fileNames, address)
		declaration.SelectionRange = rebaseRange(declaration.SelectionRange, fileNames, address)
		for index := range declaration.References {
			declaration.References[index].Range = rebaseRange(declaration.References[index].Range, fileNames, address)
		}
		declarations = append(declarations, declaration)
		scanner.result.Declarations = append(scanner.result.Declarations, declaration)
	}
	for _, diagnostic := range parsed.Diagnostics {
		if diagnostic.Subject != nil {
			rebased := rebaseRange(*diagnostic.Subject, fileNames, address)
			diagnostic.Subject = &rebased
		}
		scanner.result.Diagnostics = append(scanner.result.Diagnostics, diagnostic)
	}

	nextAncestors := cloneSet(ancestors)
	nextAncestors[modulePath] = true
	for _, declaration := range declarations {
		if declaration.Kind != hclparser.DeclarationModule || !isLocalSource(declaration.ModuleSource) {
			continue
		}
		childAddress := qualifyAddress(address, declaration.Address)
		if depth >= maxModuleDepth {
			scanner.warning(declaration.Range, "Local module depth limit reached", fmt.Sprintf("Module %q was not opened because the safe recursion limit is %d.", childAddress, maxModuleDepth))
			continue
		}
		if len(scanner.result.Modules) >= maxModuleInstances {
			scanner.warning(declaration.Range, "Local module instance limit reached", fmt.Sprintf("Module %q was not opened because the scan limit is %d instances.", childAddress, maxModuleInstances))
			continue
		}

		candidate := filepath.Clean(filepath.Join(modulePath, filepath.FromSlash(declaration.ModuleSource)))
		if !insideRoot(scanner.rootPath, candidate) {
			scanner.warning(declaration.Range, "Local module outside scan root skipped", fmt.Sprintf("Module %q resolves to %q, outside the selected root. Open a common parent directory to include it.", childAddress, candidate))
			continue
		}
		child, err := hclparser.ScanDir(candidate)
		if err != nil {
			scanner.warning(declaration.Range, "Unable to open local module", fmt.Sprintf("Module %q (%s): %v", childAddress, declaration.ModuleSource, err))
			continue
		}
		if !insideRoot(scanner.rootPath, child.Module.Path) {
			scanner.warning(declaration.Range, "Linked module outside scan root skipped", fmt.Sprintf("Module %q resolves through a symbolic link to %q, outside the selected root.", childAddress, child.Module.Path))
			continue
		}
		if nextAncestors[child.Module.Path] {
			scanner.warning(declaration.Range, "Local module cycle skipped", fmt.Sprintf("Module %q resolves to an ancestor directory and was not opened again.", childAddress))
			continue
		}
		scanner.visit(child, childAddress, address, declaration.ModuleSource, depth+1, nextAncestors)
	}
}

func (scanner *scanner) warning(subject hclparser.Range, summary, detail string) {
	scanner.result.Diagnostics = append(scanner.result.Diagnostics, hclparser.Diagnostic{
		Severity: "warning",
		Summary:  summary,
		Detail:   detail,
		Subject:  &subject,
	})
}

func (scanner *scanner) sort() {
	sort.Slice(scanner.result.Modules, func(i, j int) bool {
		return moduleSortKey(scanner.result.Modules[i].Address) < moduleSortKey(scanner.result.Modules[j].Address)
	})
	sort.Slice(scanner.result.Files, func(i, j int) bool {
		return scanner.result.Files[i].Path < scanner.result.Files[j].Path
	})
	sort.SliceStable(scanner.result.Declarations, func(i, j int) bool {
		left, right := scanner.result.Declarations[i], scanner.result.Declarations[j]
		if left.ModuleAddress != right.ModuleAddress {
			return moduleSortKey(left.ModuleAddress) < moduleSortKey(right.ModuleAddress)
		}
		if left.File != right.File {
			return left.File < right.File
		}
		return left.Range.Start.Byte < right.Range.Start.Byte
	})
}

func moduleSortKey(address string) string {
	if address == "" || address == hclparser.RootModuleAddress {
		return "\x00"
	}
	return address
}

func isLocalSource(source string) bool {
	return strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../")
}

func insideRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func qualifyAddress(moduleAddress, localAddress string) string {
	if moduleAddress == "" || moduleAddress == hclparser.RootModuleAddress {
		return localAddress
	}
	return moduleAddress + "." + localAddress
}

func instanceFile(moduleAddress, file string) string {
	if moduleAddress == "" || moduleAddress == hclparser.RootModuleAddress {
		return file
	}
	return moduleAddress + "/" + file
}

func logicalFile(names map[string]string, file, moduleAddress string) string {
	if logical, ok := names[file]; ok {
		return logical
	}
	if file == "" {
		return ""
	}
	return instanceFile(moduleAddress, filepath.Base(file))
}

func rebaseRange(value hclparser.Range, names map[string]string, moduleAddress string) hclparser.Range {
	value.File = logicalFile(names, value.File, moduleAddress)
	return value
}

func cloneSet(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source)+1)
	for key, value := range source {
		result[key] = value
	}
	return result
}
