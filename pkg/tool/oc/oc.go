package oc

import (
	"fmt"
	"os"
	"path/filepath"
)

// Tool implements the interface to manage the 'openshift-client' (aka 'oc') binary
type Tool struct{}

func NewTool() *Tool {
	return &Tool{}
}

func (t *Tool) Name() string {
	return "oc"
}

func (t *Tool) Install(rootDir string) error {
	ocDir := filepath.Join(rootDir, "oc")
	err := os.Mkdir(ocDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create install directory for oc: %w", err)
	}

	return nil
}

func (t *Tool) Configure() error {
	return nil
}

func (t *Tool) Remove() error {
	return nil
}
