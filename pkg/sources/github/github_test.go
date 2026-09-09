package github

import (
	"testing"

	"github.com/google/go-github/v51/github"
)

func assets(names ...string) []*github.ReleaseAsset {
	result := make([]*github.ReleaseAsset, 0, len(names))
	for _, n := range names {
		name := n
		result = append(result, &github.ReleaseAsset{Name: &name})
	}
	return result
}

func TestFindChecksumAsset(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		assets []*github.ReleaseAsset
		want   []string
	}{
		"legacy checksums.txt naming (rosa <= v1.2.60)": {
			assets: assets("rosa_1.2.60_checksums.txt", "rosa_Linux_x86_64.tar.gz"),
			want:   []string{"rosa_1.2.60_checksums.txt"},
		},
		"new SHA256SUMS naming (rosa >= v1.2.65)": {
			assets: assets("rosa_1.2.65_SHA256SUMS", "rosa_linux_amd64.zip"),
			want:   []string{"rosa_1.2.65_SHA256SUMS"},
		},
		"SHA256SUMS manifest alongside detached signature": {
			assets: assets("rosa_1.2.65_SHA256SUMS", "rosa_1.2.65_SHA256SUMS.sig", "rosa_linux_amd64.zip"),
			want:   []string{"rosa_1.2.65_SHA256SUMS"},
		},
		"sha256sum.txt naming": {
			assets: assets("sha256sum.txt", "tool_linux_amd64.tar.gz"),
			want:   []string{"sha256sum.txt"},
		},
		"no checksum asset present": {
			assets: assets("tool_linux_amd64.tar.gz"),
			want:   nil,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := FindChecksumAsset(tc.assets)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d matches, want %d: %v", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				if got[i].GetName() != w {
					t.Errorf("match %d: got %q, want %q", i, got[i].GetName(), w)
				}
			}
		})
	}
}
