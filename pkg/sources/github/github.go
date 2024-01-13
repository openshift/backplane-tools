package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/google/go-github/v51/github"
	"github.com/openshift/backplane-tools/pkg/utils"
	"golang.org/x/oauth2"
)

type Source struct {
	// Owner specifies the organization or user this tool belongs to
	Owner string

	// Repo specifies the repository of the tool
	Repo string

	// client is used to interact with GitHub
	client *github.Client
}

func NewSource(owner, repo string) *Source {
	token, _ := auth.TokenForHost("")
	var tc *http.Client
	if token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(ctx, ts)
	} else {
		tc = nil
	}
	tool := &Source{
		Owner:  owner,
		Repo:   repo,
		client: github.NewClient(tc),
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

// FetchTags returns the latest tag
// FetchLatestTag returns the latest tag
func (s Source) FetchLatestTag() (string, error) {
	ctx := context.Background()
	tags, _, err := s.client.Repositories.ListTags(ctx, s.Owner, s.Repo, nil)
	if err != nil {
		return "", err
	}
	if len(tags) > 0 {
		return *tags[0].Name, nil
	}
	return "", nil
}

// DownloadReleaseAssets downloads the provided GitHub release assets and stores them in the given directory.
// The resulting files will match the assets' names
func (s Source) DownloadReleaseAssets(assets []*github.ReleaseAsset, dir string) error {
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

func (s Source) downloadReleaseAsset(asset *github.ReleaseAsset, dir string) error {
	// Per the documentation for this method (https://pkg.go.dev/github.com/google/go-github/v51/github#RepositoriesService.DownloadReleaseAsset),
	// a redirectURL will not be returned if an http.Client is provided for the followRedirectsClient argument.
	reader, _, err := s.client.Repositories.DownloadReleaseAsset(context.TODO(), s.Owner, s.Repo, asset.GetID(), s.client.Client())
	if err != nil {
		return err
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			panic(fmt.Sprintf("failed to close reader from GitHub asset '%s'", asset.GetName()))
		}
	}()
	filePath := filepath.Join(dir, asset.GetName())

	return utils.WriteFile(reader, filePath, 0o755)
}

// FindAssetsForOS searches the provided list of assets and returns the subset, if any, matching
// the local OS as defined by runtime.GOOS, as well as any well-known alternative names for the OS
func FindAssetsForOS(assets []*github.ReleaseAsset) []*github.ReleaseAsset {
	matches := []*github.ReleaseAsset{}
	for _, asset := range assets {
		if utils.ContainsAny(strings.ToLower(asset.GetName()), utils.GetOSAliases()) {
			matches = append(matches, asset)
		}
	}
	return matches
}

// FindAssetsForArch searches the provided list of assets and returns the subset, if any, matching
// the local architecture as defined by runtime.GOARCH, as well as well-known alternative names for the
// architecture
func FindAssetsForArch(assets []*github.ReleaseAsset) []*github.ReleaseAsset {
	matches := []*github.ReleaseAsset{}
	for _, asset := range assets {
		if utils.ContainsAny(strings.ToLower(asset.GetName()), utils.GetArchAliases()) {
			matches = append(matches, asset)
		}
	}
	return matches
}

// FindAssetsForArchAndOS searches the provided list of assets and returns the subset, if any, matching
// the local architecture and OS, as defined by runtime.GOARCH and runtime.GOOS, respectively.
// In addition to these values, well-known alternatives are also used when searching.
func FindAssetsForArchAndOS(assets []*github.ReleaseAsset) []*github.ReleaseAsset {
	return FindAssetsForOS(FindAssetsForArch(assets))
}

// FindAssetMatching searches the provided slice of assets for entries whose Name matches the given pattern.
// All matches are returned. If no matches are found, an error is returned.
func FindAssetsMatching(pattern string, assets []*github.ReleaseAsset) ([]*github.ReleaseAsset, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return []*github.ReleaseAsset{}, fmt.Errorf("provided pattern '%s' could not be compiled as regex: %w", pattern, err)
	}

	matches := []*github.ReleaseAsset{}
	for _, asset := range assets {
		if re.MatchString(asset.GetName()) {
			matches = append(matches, asset)
		}
	}
	if len(matches) == 0 {
		return []*github.ReleaseAsset{}, fmt.Errorf("failed to find asset matching '%s'", pattern)
	}
	return matches, nil
}

// FindAssetContaining searches the provided slice of assets for entries whose Name contains all of the
// given search terms.
func FindAssetsContaining(terms []string, assets []*github.ReleaseAsset) []*github.ReleaseAsset {
	matches := []*github.ReleaseAsset{}
	for _, asset := range assets {
		if utils.ContainsAll(asset.GetName(), terms) {
			matches = append(matches, asset)
		}
	}
	return matches
}

// FindAssetsExcluding searches the provided slice of assets for entries whose Name contains none of the
// given search terms.
func FindAssetsExcluding(terms []string, assets []*github.ReleaseAsset) []*github.ReleaseAsset {
	matches := []*github.ReleaseAsset{}
	for _, asset := range assets {
		if !utils.ContainsAny(asset.GetName(), terms) {
			matches = append(matches, asset)
		}
	}
	return matches
}
