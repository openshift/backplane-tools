package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/github"
)

type Source struct {
	// Owner specifies the organization or user this tool belongs to
	Owner string

	// Repo specifies the repository of the tool
	Repo string

	// client is used to interact with GitHub
	client *github.Client
}

func NewGithubSource(owner, repo string) *Source {
	tool := &Source{
		Owner:  owner,
		Repo:   repo,
		client: github.NewClient(nil),
	}
	return tool
}

// ListReleases returns all releases of the tool from GitHub
func (s Source) ListReleases(opts *github.ListOptions) ([]*github.RepositoryRelease, error) {
	releases, response, err := s.client.Repositories.ListReleases(context.TODO(), s.Owner, s.Repo, opts)
	if err != nil {
		return []*github.RepositoryRelease{}, err
	}
	err = github.CheckResponse(response.Response)
	if err != nil {
		return []*github.RepositoryRelease{}, err
	}
	return releases, nil
}

// FetchRelease returns the specified release of the tool from GitHub
func (s Source) FetchRelease(releaseID int64) (*github.RepositoryRelease, error) {
	release, response, err := s.client.Repositories.GetRelease(context.TODO(), s.Owner, s.Repo, releaseID)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}
	err = github.CheckResponse(response.Response)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}
	return release, nil
}

// FetchLatestRelease returns the latest release of the tool from GitHub
func (s Source) FetchLatestRelease() (*github.RepositoryRelease, error) {
	release, response, err := s.client.Repositories.GetLatestRelease(context.TODO(), s.Owner, s.Repo)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}
	err = github.CheckResponse(response.Response)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}
	return release, nil
}

// DownloadReleaseAssets downloads the provided GitHub release assets and stores them in the given directory.
// The resulting files will match the assets' names
func (s Source) DownloadReleaseAssets(assets []github.ReleaseAsset, dir string) error {
	var downloadErrors []error
	for _, asset := range assets {
		err := s.downloadReleaseAsset(asset, dir)
		if err != nil {
			downloadErrors = append(downloadErrors, err)
		}
	}
	if len(downloadErrors) == 0 {
		return nil
	}

	return errors.Join(downloadErrors...)
}

func (s Source) downloadReleaseAsset(asset github.ReleaseAsset, dir string) error {
	reader, redirectURL, err := s.client.Repositories.DownloadReleaseAsset(context.TODO(), s.Owner, s.Repo, asset.GetID())
	if err != nil {
		return err
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			panic(fmt.Sprintf("failed to close reader from GitHub asset '%s'", asset.GetName()))
		}
	}()
	if redirectURL != "" {
		resp, err := http.Get(redirectURL)
		if err != nil {
			return err
		}
		reader = resp.Body
	}
	filePath := filepath.Join(dir, asset.GetName())
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = os.Chmod(file.Name(), os.FileMode(0755))
	if err != nil {
		return err
	}
	_, err = file.ReadFrom(reader)
	if err != nil {
		return err
	}
	return nil
}
