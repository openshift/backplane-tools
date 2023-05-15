package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	gogithub "github.com/google/go-github/v51/github"
)

// testServers emulate GitHup API v3 responses for consistent testing
// Refer to https://github.com/google/go-github/blob/master/github/github_test.go#L37
// and https://docs.github.com/en/rest?apiVersion=2022-11-28 for guidance on implementation
type testServer struct {
	server *httptest.Server
	mux *http.ServeMux
}

// newTestServer constructs a testServer object
func newTestServer() *testServer {
	m := http.NewServeMux()
	s := httptest.NewServer(m)
	server := &testServer {
		server: s,
		mux: m,
	}
	return server
}

// cleanup handles all post-test actions required to clean up a testServer
func (t *testServer) cleanup() {
	t.server.Close()
}

// url formats the testServer's URL and returns the equivalent url.URL object
func (t *testServer) url() (*url.URL, error) {
	// NOTE: go-github requires that the testServer's URL ends with a '/'
	return url.Parse(fmt.Sprintf("%s/",t.server.URL))
}

// addArbitraryHandler adds an arbitrary handler function for the provided path
func (t *testServer) addArbitraryHandler(path string, handler http.HandlerFunc) {
	t.mux.HandleFunc(path, handler)
}

// addListReleasesHandler adds a handler function for go-github's RepositoriesService.ListReleases() function
func (t *testServer) addListReleasesHandler(owner string, repo string, handler http.HandlerFunc) {
	t.mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/releases", owner, repo), handler)
}

// addFetchReleaseHandler adds a handler function for go-github's RepositoriesService.FetchRelease() function
func (t *testServer) addFetchReleaseHandler(owner string, repo string, release int64, handler http.HandlerFunc) {
	t.mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/releases/%d", owner, repo, release), handler)
}

// addFetchLatestReleaseHandler adds a handler function for go-github's RepositoriesService.FetchLatestRelease() function
func (t *testServer) addFetchLatestReleaseHandler(owner string, repo string, handler http.HandlerFunc) {
	t.mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo), handler)
}

// addDownloadReleaseAssetHandler adds a handler function for go-github's RepositoriesService.DownloadReleaseAsset() function
func (t *testServer) addDownloadReleaseAssetHandler(owner string, repo string, asset int64, handler http.HandlerFunc) {
	t.mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/releases/assets/%d", owner, repo, asset), handler)
}

// TestSource directs a normal Source object's requests to an httptest server for more predictable
// and consistent testing
type TestSource struct {
	*Source
	server *testServer
}

// NewTestSource constructs a TestSource
func NewTestSource(owner, repo string) (*TestSource, error) {
	server := newTestServer()
	serverURL, err := server.url()
	if err != nil {
		return &TestSource{}, fmt.Errorf("failed to parse server URL: %w", err)
	}
	src := NewSource(owner, repo)
	src.client.BaseURL = serverURL
	ts := &TestSource{
		Source: src,
		server: server,
	}
	return ts, nil
}

// Cleanup handles all post-test actions required to clean up a TestSource
func (t *TestSource) Cleanup() {
	t.server.cleanup()
}

// buildHandlerForResponse creates a consistent handler function for the provided response.
// Responses are Marshal()'d to json prior to being written
func (t *TestSource) buildHandlerForResponse(resp interface{}) (http.HandlerFunc, error) {
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {}, err
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(respBytes))
	}
	return handler, nil
}

// buildHandlerForRaw creates a consistent handler function for the provided response.
// Responses are not modified prior to being written
func (t *TestSource) buildHandlerForRaw(resp interface{}) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, resp)
	}
	return handler
}

// AddListReleasesResponse allows tests to specify the response they expect from the TestSource
// when calling ListReleases()
func (t *TestSource) AddListReleasesResponse(resp []gogithub.RepositoryRelease) error {
	handler, err := t.buildHandlerForResponse(resp)
	if err != nil {
		return err
	}
	t.server.addListReleasesHandler(t.Owner, t.Repo, handler)
	return nil
}

// AddFetchReleaseResponse allows tests to specify the response they expect from the TestSource
// when calling FetchRelease()
func (t *TestSource) AddFetchReleaseResponse(resp gogithub.RepositoryRelease) error {
	if resp.ID == nil {
		return fmt.Errorf("cannot add FetchRelease response: provided RepositoryRelease has no ID defined")
	}
	handler, err := t.buildHandlerForResponse(resp)
	if err != nil {
		return err
	}
	t.server.addFetchReleaseHandler(t.Owner, t.Repo, resp.GetID(), handler)
	return nil
}

// AddFetchLatestReleaseResponse allows tests to specify the response they expect from the TestSource
// when calling FetchLatestRelease()
func (t *TestSource) AddFetchLatestReleaseResponse(resp gogithub.RepositoryRelease) error {
	handler, err := t.buildHandlerForResponse(resp)
	if err != nil {
		return err
	}
	t.server.addFetchLatestReleaseHandler(t.Owner, t.Repo, handler)
	return nil
}

// AddResponse allows tests to specify the response they expect for the provided path
func (t *TestSource) AddResponse(path string, handler http.HandlerFunc) {
	t.server.addArbitraryHandler(path, handler)
}

// AddDownloadReleaseAssetResponse allows tests to specify the contents they expect the provided asset to have
// when calling DownloadReleaseAsset()
func (t *TestSource) AddDownloadReleaseAssetResponse(asset *gogithub.ReleaseAsset, resp interface{}) error {
	if asset.ID == nil {
		return fmt.Errorf("cannot add DownloadReleaseAsset response: provided ReleaseAsset has no ID defined")
	}
	handler := t.buildHandlerForRaw(resp)
	t.server.addDownloadReleaseAssetHandler(t.Owner, t.Repo, asset.GetID(), handler)
	return nil
}
