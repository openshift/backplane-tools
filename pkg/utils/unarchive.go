package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Unarchive decompresses and extracts the contents of .tar.gz bundles to the specified destination
func Unarchive (source string, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open tarball '%s': %v", source, err)
	}
	defer func() {
		err = src.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close '%s': %v\n", src.Name(), err)
		}
	}()
	uncompressed, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to read the gzip file '%s': %v", source, err)
	}
	defer func() {
		err = uncompressed.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close gzip file '%s': %v", source, err)
		}
	}()
	arc := tar.NewReader(uncompressed)
	var f *tar.Header
	for {
		f, err = arc.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read from archive '%s': %v", source, err)
		}
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(filepath.Join(destination, f.Name), f.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed to create a directory : %v", err)
			}
		} else {
			err = extractFile(destination, f, arc)
			if err != nil {
				return fmt.Errorf("failed to extract files: %v", err)
			}
		}
	}
	return nil
}

func extractFile(destination string, f *tar.Header, arc io.Reader) error {
	dst, err := os.Create(filepath.Join(destination, f.Name))
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer func() {
		err = dst.Close()
		if err != nil {
			fmt.Printf("warning: failed to close '%s': %v\n", dst.Name(), err)
		}
	}()

	err = dst.Chmod(os.FileMode(f.Mode))
	if err != nil {
		return fmt.Errorf("failed to set permission on '%s': %v", dst.Name(), err)
	}
	_, err = dst.ReadFrom(arc)
	if err != nil {
		return fmt.Errorf("failed to read from archive  %v", err)
	}
	return nil
}
