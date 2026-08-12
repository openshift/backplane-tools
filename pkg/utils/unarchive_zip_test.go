package utils

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestUnzip_ValidArchive(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "valid.zip")
	dest := t.TempDir()

	if err := writeZip(source, map[string]string{
		"tool": "zip contents",
	}); err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	if err := Unzip(source, dest); err != nil {
		t.Fatalf("Unzip failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "tool"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != "zip contents" {
		t.Fatalf("got %q, want %q", string(data), "zip contents")
	}
}

func TestUnzip_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "evil.zip")
	dest := t.TempDir()
	outside := filepath.Join(filepath.Dir(dest), "zip-slip-outside.txt")

	if err := writeZip(source, map[string]string{
		"../../zip-slip-outside.txt": "pwned",
	}); err != nil {
		t.Fatalf("failed to create malicious zip: %v", err)
	}

	err := Unzip(source, dest)
	if err == nil {
		t.Fatal("expected Unzip to reject path traversal")
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		os.Remove(outside)
		t.Fatal("path traversal wrote outside destination directory")
	}
}

func writeZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}
