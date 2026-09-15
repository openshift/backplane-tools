package rosa

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	gogithub "github.com/google/go-github/v51/github"
)

func TestFindChecksumAsset(t *testing.T) {
	tests := []struct {
		name       string
		assetNames []string
		want       string
		wantError  bool
	}{
		{
			name:       "legacy checksums file",
			assetNames: []string{"rosa_Darwin_arm64.tar.gz", "rosa_1.2.64_checksums.txt"},
			want:       "rosa_1.2.64_checksums.txt",
		},
		{
			name:       "sha256sums file",
			assetNames: []string{"rosa_darwin_arm64.zip", "rosa_1.2.65_SHA256SUMS", "rosa_1.2.65_SHA256SUMS.sig"},
			want:       "rosa_1.2.65_SHA256SUMS",
		},
		{
			name:       "missing checksum file",
			assetNames: []string{"rosa_darwin_arm64.zip"},
			wantError:  true,
		},
		{
			name:       "ambiguous checksum files",
			assetNames: []string{"rosa_checksums.txt", "rosa_SHA256SUMS"},
			wantError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assets := make([]*gogithub.ReleaseAsset, 0, len(test.assetNames))
			for _, name := range test.assetNames {
				assets = append(assets, &gogithub.ReleaseAsset{Name: gogithub.String(name)})
			}

			got, err := findChecksumAsset(assets)
			if test.wantError {
				if err == nil {
					t.Fatal("findChecksumAsset succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("findChecksumAsset failed: %v", err)
			}
			if got.GetName() != test.want {
				t.Fatalf("findChecksumAsset returned %q, want %q", got.GetName(), test.want)
			}
		})
	}
}

func TestExtractArchiveRejectsUnknownFormat(t *testing.T) {
	err := extractArchive("rosa.exe", t.TempDir())
	if err == nil {
		t.Fatal("extractArchive succeeded for an unsupported format")
	}
}

func TestExtractArchiveZip(t *testing.T) {
	source := filepath.Join(t.TempDir(), "rosa.zip")
	file, err := os.Create(source)
	if err != nil {
		t.Fatalf("failed to create ZIP: %v", err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("rosa")
	if err != nil {
		t.Fatalf("failed to create ZIP entry: %v", err)
	}
	if _, err := entry.Write([]byte("binary contents")); err != nil {
		t.Fatalf("failed to write ZIP entry: %v", err)
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("failed to close ZIP: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close ZIP file: %v", err)
	}

	destination := t.TempDir()
	if err := extractArchive(source, destination); err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(destination, "rosa"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(contents) != "binary contents" {
		t.Fatalf("extracted %q, want %q", contents, "binary contents")
	}
}
