package hclparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirFindsTerraformDeclarations(t *testing.T) {
	result, err := ScanDir(filepath.Join("..", "..", "testdata", "simple-module"))
	if err != nil {
		t.Fatalf("ScanDir() error = %v", err)
	}

	if result.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %q, want %q", result.SchemaVersion, SchemaVersion)
	}
	if got, want := len(result.Files), 2; got != want {
		t.Fatalf("files = %d, want %d", got, want)
	}
	if got, want := len(result.Diagnostics), 0; got != want {
		t.Fatalf("diagnostics = %d, want %d: %#v", got, want, result.Diagnostics)
	}

	wantAddresses := []string{
		"module.labels",
		"var.project",
		"local.full_name",
		"local.enabled",
		"data.aws_caller_identity.current",
		"aws_s3_bucket.assets",
		"output.bucket_id",
	}
	if got, want := len(result.Declarations), len(wantAddresses); got != want {
		t.Fatalf("declarations = %d, want %d: %#v", got, want, result.Declarations)
	}
	for index, want := range wantAddresses {
		declaration := result.Declarations[index]
		if declaration.Address != want {
			t.Errorf("declaration[%d].Address = %q, want %q", index, declaration.Address, want)
		}
		if declaration.Range.Start.Line < 1 || declaration.Range.End.Line < declaration.Range.Start.Line {
			t.Errorf("declaration[%d] has invalid range: %#v", index, declaration.Range)
		}
		if declaration.SelectionRange.Start.Byte < declaration.Range.Start.Byte || declaration.SelectionRange.End.Byte > declaration.Range.End.Byte {
			t.Errorf("declaration[%d] selection is outside declaration: %#v", index, declaration)
		}
	}
}

func TestScanDirKeepsDiagnosticsForMalformedFile(t *testing.T) {
	result, err := ScanDir(filepath.Join("..", "..", "testdata", "malformed"))
	if err != nil {
		t.Fatalf("ScanDir() error = %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("diagnostics is empty, want syntax error")
	}
	if result.Diagnostics[0].Severity != "error" {
		t.Fatalf("diagnostic severity = %q, want error", result.Diagnostics[0].Severity)
	}
	if result.Diagnostics[0].Subject == nil {
		t.Fatal("diagnostic subject is nil, want source range")
	}
}

func TestScanDirSkipsTerraformSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "outside.tf")
	if err := os.WriteFile(target, []byte(`variable "secret" {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "linked.tf")); err != nil {
		t.Skipf("symlink is unavailable: %v", err)
	}

	result, err := ScanDir(root)
	if err != nil {
		t.Fatalf("ScanDir() error = %v", err)
	}
	if len(result.Files) != 0 || len(result.Declarations) != 0 {
		t.Fatalf("symlink was scanned: %#v", result)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Severity != "warning" {
		t.Fatalf("diagnostics = %#v, want one warning", result.Diagnostics)
	}
}

func TestScanDirRejectsFilePath(t *testing.T) {
	_, err := ScanDir(filepath.Join("..", "..", "testdata", "simple-module", "network.tf"))
	if err == nil {
		t.Fatal("ScanDir() error = nil, want not-a-directory error")
	}
}
