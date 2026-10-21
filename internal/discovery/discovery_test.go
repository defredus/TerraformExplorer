package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/terraform-explorer/terraform-explorer/internal/hclparser"
)

func TestScanDirDiscoversNestedLocalModuleInstances(t *testing.T) {
	result, err := ScanDir(filepath.Join("..", "..", "testdata", "nested-project"))
	if err != nil {
		t.Fatal(err)
	}

	wantModules := map[string]bool{
		hclparser.RootModuleAddress:    true,
		"module.service":               true,
		"module.service.module.labels": true,
	}
	if len(result.Modules) != len(wantModules) {
		t.Fatalf("modules = %#v, want %d instances", result.Modules, len(wantModules))
	}
	for _, module := range result.Modules {
		if !wantModules[module.Address] {
			t.Errorf("unexpected module instance %q", module.Address)
		}
	}
	if got, want := len(result.Files), 3; got != want {
		t.Fatalf("files = %d, want %d: %#v", got, want, result.Files)
	}

	assertDeclaration(t, result, "module.service", "var.name")
	assertDeclaration(t, result, "module.service.module.labels", "local.value")
	for _, module := range result.Modules {
		if module.Address == "module.remote" {
			t.Fatal("external registry module was opened")
		}
	}
}

func TestScanDirSkipsLocalModuleOutsideSelectedRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	outside := filepath.Join(parent, "outside")
	mustWrite(t, filepath.Join(root, "main.tf"), `module "outside" { source = "../outside" }`)
	mustWrite(t, filepath.Join(outside, "main.tf"), `variable "secret" {}`)

	result, err := ScanDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 1 {
		t.Fatalf("modules = %#v, want only root", result.Modules)
	}
	assertDiagnostic(t, result, "outside scan root")
}

func TestScanDirStopsLocalModuleCycles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.tf"), `module "child" { source = "./child" }`)
	mustWrite(t, filepath.Join(root, "child", "main.tf"), `module "root_again" { source = "../" }`)

	result, err := ScanDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 2 {
		t.Fatalf("modules = %#v, want root and child", result.Modules)
	}
	assertDiagnostic(t, result, "cycle skipped")
}

func assertDeclaration(t *testing.T, result hclparser.Result, moduleAddress, address string) {
	t.Helper()
	for _, declaration := range result.Declarations {
		if declaration.ModuleAddress == moduleAddress && declaration.Address == address {
			return
		}
	}
	t.Errorf("declaration %s in %s was not found", address, moduleAddress)
}

func assertDiagnostic(t *testing.T, result hclparser.Result, text string) {
	t.Helper()
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(strings.ToLower(diagnostic.Summary), strings.ToLower(text)) {
			return
		}
	}
	t.Errorf("diagnostic containing %q was not found: %#v", text, result.Diagnostics)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
