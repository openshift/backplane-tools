package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/openshift/backplane-tools/pkg/utils"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Source struct {
	// bucket defines the name of the bucket to retrieve files from
	bucketName string
	// Client defines the component which will retrieve files from a gcloud bucket
	client storage.Client
}

// NewSource creates a Source given the google cloud bucket's name
func NewSource(bucketName string) (*Source, error) {
	client, err := storage.NewClient(context.TODO(), option.WithoutAuthentication())
	if err != nil {
		return &Source{}, err
	}
	s := &Source{
		bucketName: bucketName,
		client:     *client,
	}
	return s, nil
}

func (s *Source) bucket() *storage.BucketHandle {
	return s.client.Bucket(s.bucketName)
}

// ListObjects fetches all objects in the Source's bucket matching the provided prefix
// Objects are returned in lexigraphical order
func (s *Source) ListObjects(prefix string) ([]*storage.ObjectAttrs, error) {
	query := storage.Query{
		Prefix: prefix,
	}
	err := query.SetAttrSelection([]string{"Name"})
	if err != nil {
		return []*storage.ObjectAttrs{}, fmt.Errorf("failed to set attribute selection for query: %w", err)
	}
	it := s.bucket().Objects(context.TODO(), &query)

	objs := []*storage.ObjectAttrs{}
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return []*storage.ObjectAttrs{}, fmt.Errorf("error while listing bucket objects: %w", err)
		}
		objs = append(objs, attrs)
	}
	return objs, nil
}

func (s *Source) DownloadObject(obj *storage.ObjectAttrs, dir string) error {
	objReader, err := s.bucket().Object(obj.Name).NewReader(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to create reader for object '%s': %w", obj.Name, err)
	}

	filePath := filepath.Join(dir, obj.Name)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file '%s': %w", filePath, err)
	}

	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to close '%s': %v\n", filePath, closeErr)
		}
	}()

	err = file.Chmod(0o755)
	if err != nil {
		return fmt.Errorf("failed to set permission on '%s': %w", filePath, err)
	}

	_, err = file.ReadFrom(objReader)
	if err != nil {
		return fmt.Errorf("failed to read object '%s' from bucket '%s': %w", obj.Name, s.bucketName, err)
	}
	return nil
}

// FindObjectsForOS searches the provided list of objects and returns the subset, if any, whose name
// contains references to the local system's OS, as defined by runtime.GOOS, in addition to
// any well-known alternative names for the architecture
func (s *Source) FindObjectsForOS(objs []*storage.ObjectAttrs) []*storage.ObjectAttrs {
	matches := []*storage.ObjectAttrs{}
	for _, obj := range objs {
		if utils.ContainsAny(strings.ToLower(obj.Name), utils.GetOSAliases()) {
			matches = append(matches, obj)
		}
	}
	return matches
}

// FindObjectsForArch searches the provided list of objects and returns the subset, if any, whose name
// contains references to the local system's architecture, as defined by runtime.GOARCH, in addition to
// any well-known alternative names for the architecture
func (s *Source) FindObjectsForArch(objs []*storage.ObjectAttrs) []*storage.ObjectAttrs {
	matches := []*storage.ObjectAttrs{}
	for _, obj := range objs {
		if utils.ContainsAny(strings.ToLower(obj.Name), utils.GetArchAliases()) {
			matches = append(matches, obj)
		}
	}
	return matches
}

// FindObjectsForArchAndOS searches the provided list of assets and returns the subset, if any, matching
// the local architecture and OS, as defined by runtime.GOARCH and runtime.GOOS, respectively. In addition
// to these values, well-known alternatives are also used when searching.
func (s *Source) FindObjectsForArchAndOS(objs []*storage.ObjectAttrs) []*storage.ObjectAttrs {
	return s.FindObjectsForOS(s.FindObjectsForArch(objs))
}

func (s *Source) FindLatest(objs []*storage.ObjectAttrs) *storage.ObjectAttrs {
	names := []string{}
	for _, obj := range objs {
		names = append(names, obj.Name)
	}
	sort.Strings(names)
	latest := names[len(names)-1]
	for _, obj := range objs {
		if obj.Name == latest {
			return obj
		}
	}
	panic(fmt.Sprintf("cannot find previously present object %s. All storage objects: %#v", latest, objs))
}
