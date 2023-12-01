package base

import (
	"fmt"
	"strings"

	"github.com/openshift/backplane-tools/pkg/sources/openshift/mirror"
	"github.com/openshift/backplane-tools/pkg/utils"
)

type Mirror struct {
	Default
	Source   *mirror.Source
	BaseSlug string
}

// LatestVersion retrieves the version info contained within the provided release.txt file
func (t *Mirror) _LatestVersion() (string, error) {
	// Retrieve latest release info to determine which version we're operating on
	releaseSlug := fmt.Sprintf("%s/release.txt", t.BaseSlug)
	releaseData, err := t.Source.GetFileContents(releaseSlug)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve release info from %s: %w", releaseSlug, err)
	}
	defer func() {
		closeErr := releaseData.Close()
		if closeErr != nil {
			fmt.Printf("WARNING: failed to close response body: %v\n", closeErr)
		}
	}()

	line, err := utils.GetLineInReader(releaseData, "Version:")
	if err != nil {
		return "", fmt.Errorf("failed to determine version info from release file: %w", err)
	}

	tokens := strings.Fields(line)
	if len(tokens) != 2 {
		return "", fmt.Errorf("failed to parse version info from release: expected 2 tokens, got %d.\nVersion info retrieved:\n%s", len(tokens), line)
	}
	if tokens[0] != "Version:" {
		return "", fmt.Errorf("failed to parse version info from release: expected line to begin with 'Version:', got '%s'.\nVersion info retrieved:\n%s", tokens[0], line)
	}
	return tokens[1], nil
}

func (t *Mirror) LatestVersion() (string, error) {
	if t.latestVersion == "" {
		version, err := t._LatestVersion()
		if err != nil {
			return "", err
		}
		t.latestVersion = version
	}
	return t.latestVersion, nil
}
