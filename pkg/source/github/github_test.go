package github

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/v51/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_Source(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github Source Package")
}

var _ = Context("Using a GitHub Source:", func() {
	var (
		src *TestSource
		err error
	)
	
	// Global setup for all Source tests
	BeforeEach(func() {
		src, err = NewTestSource("fake", "test")
		Expect(err).ToNot(HaveOccurred())
	})

	// Global cleanup for all Source tests
	AfterEach(func() {
		src.Cleanup()
	})

	// Source tests
	When("ListReleases() is called and", func() {
		When("there is at lease one release for the repository", func() {
			It("returns the repository's releases without error", func() {
				// Setup
				release1ID := int64(123)
				release2ID := int64(456)
				release3ID := int64(789)
				releases := []gogithub.RepositoryRelease{
					{
						ID: &release1ID,
					},
					{
						ID: &release2ID,
					},
					{
						ID: &release3ID,
					},
				}
				err = src.AddListReleasesResponse(releases)
				Expect(err).ToNot(HaveOccurred())

				// Run test
				results, err := src.ListReleases(&gogithub.ListOptions{})

				// Validate results
				Expect(err).ToNot(HaveOccurred())
				Expect(len(results)).To(Equal(len(releases)))
				for _, result := range results {
					Expect(releases).To(ContainElement(*result))
				}
			})
		})
		When("ListOptions are passed", func() {
			It("includes them when listing releases", func() {
				// Setup
				opts := gogithub.ListOptions{
					Page: 2,
				}

				// Formulate custom response to check that option values are being respected
				// If we don't see the page defined in the request's query, then the provided
				// options aren't being respected
				src.server.addListReleasesHandler(src.Owner, src.Repo, func(w http.ResponseWriter, r *http.Request) {
					if !strings.Contains(r.RequestURI, "page=2") {
						http.NotFound(w, r)
						return
					}
					// Why net/http includes a built-in "NotFound()" to return a 404, but not a "OK()" is beyond me
					w.Header().Set("status", fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)))
				})

				// Run test
				_, err := src.ListReleases(&opts)

				// Validate results
				Expect(err).ToNot(HaveOccurred())
			})
		})
		When("the go-github client returns an error", func() {
			It("is returned so it can be handled", func() {
				// Setup
				// Reset the BaseURL to force an error
				src.client.BaseURL, err = url.Parse("invalidurl")
				Expect(err).ToNot(HaveOccurred())
				
				// Run test
				_, err := src.ListReleases(&gogithub.ListOptions{})

				// Validate results
				Expect(err).To(HaveOccurred())
			})
		})
		When("the GitHub API returns a non-200 HTTP code", func() {
			It("returns an error to handle", func() {
				// No setup required - by not providing a valid
				// response from the TestSource's internal server,
				// the HTTP code will be 404
				// Run test
				_, err := src.ListReleases(&gogithub.ListOptions{})

				// Validate results
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("FetchRelease() is called and", func() {
		releaseTag := int64(123456)
		When("the provided release exists", func() {
			It("returns the release without error", func() {
				// Setup
				respTag := "test"
				resp := gogithub.RepositoryRelease {
					TagName: &respTag,
					ID: &releaseTag,
				}
				err = src.AddFetchReleaseResponse(resp)
				Expect(err).ToNot(HaveOccurred())

				// Run test
				result, err := src.FetchRelease(releaseTag)

				// Validate results
				Expect(err).ToNot(HaveOccurred())
				Expect(*result).To(Equal(resp))
			})
		})
		When("the provided release does not exist", func() {
			It("returns an error to handle", func() {
				// Run Test
				_, err := src.FetchRelease(releaseTag)

				// Validate results
				Expect(err).To(HaveOccurred())
			})
		})
		When("the go-github client returns an error", func() {
			It("is returned so it can be handled", func() {
				// Setup
				// Reset the BaseURL to force an error
				src.client.BaseURL, err = url.Parse("invalidurl")
				Expect(err).ToNot(HaveOccurred())
				
				// Run test
				_, err := src.FetchRelease(releaseTag)

				// Validate results
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("FetchLatestRelease() is called and", func() {
		When("at lease one release exists", func() {
			It("returns the release without error", func() {
				// Setup
				respTag := "test"
				resp := gogithub.RepositoryRelease {
					TagName: &respTag,
				}
				src.AddFetchLatestReleaseResponse(resp)

				// Run test
				result, err := src.FetchLatestRelease()

				// Validate results
				Expect(err).ToNot(HaveOccurred())
				Expect(*result).To(Equal(resp))
			})
		})
		When("no releases exist for the source", func() {
			It("returns an error to handle", func() {
				// Run test
				_, err := src.FetchLatestRelease()

				// Validate results
				Expect(err).To(HaveOccurred())
			})
		})
		When("the go-github client returns an error", func() {
			It("is returned so it can be handled", func() {
				// Setup
				// Reset the BaseURL to force an error
				src.client.BaseURL, err = url.Parse("invalidurl")
				Expect(err).ToNot(HaveOccurred())
				
				// Run test
				_, err := src.FetchLatestRelease()

				// Validate results
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("DownloadReleaseAssets() is called and", func() {
		var (
			asset1, asset2, asset3 *gogithub.ReleaseAsset

			asset1Tag = int64(123)
			asset2Tag = int64(456)
			asset3Tag = int64(789)

			asset1Name = "asset1"
			asset2Name = "asset2"
			asset3Name = "asset3"

			assets []*gogithub.ReleaseAsset

			tmp string
		)
		BeforeEach(func() {
			// Create each test asset, with the contents of each asset being set as their name
			asset1 = &gogithub.ReleaseAsset{
				Name: &asset1Name,
				ID: &asset1Tag,
			}
			err = src.AddDownloadReleaseAssetResponse(asset1, asset1Name)
			Expect(err).ToNot(HaveOccurred())

			asset2 = &gogithub.ReleaseAsset{
				Name: &asset2Name,
				ID: &asset2Tag,
			}
			err = src.AddDownloadReleaseAssetResponse(asset2, asset2Name)
			Expect(err).ToNot(HaveOccurred())

			asset3 = &gogithub.ReleaseAsset{
				Name: &asset3Name,
				ID: &asset3Tag,
			}
			err = src.AddDownloadReleaseAssetResponse(asset3, asset3Name)
			Expect(err).ToNot(HaveOccurred())

			// Create asset slice and tmp dir for the following tests
			assets = []*gogithub.ReleaseAsset{asset1, asset2, asset3}

			tmp, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).ToNot(HaveOccurred())
		})
		// Clean the tmp dir and un-populate the assets slice after each test to avoid cross-contamination and memory leaks
		// (Note: Changes to src are automatically handled at the Context level, so no direct cleanup required)
		AfterEach(func() {
			err = os.RemoveAll(tmp)
			Expect(err).ToNot(HaveOccurred())

			assets = []*gogithub.ReleaseAsset{}
		})

		// checkTmpDir verifies that asset1, asset2, and asset3 exist in the tmp dir in a valid state
		var checkTmpDir = func() {
			assetDirEntries, err := os.ReadDir(tmp)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(assetDirEntries)).To(Equal(len(assets)))
			for _, assetDirEntry := range assetDirEntries {
				Expect(assetDirEntry.Type().IsRegular()).To(BeTrue())

				contentBytes, err := ioutil.ReadFile(filepath.Join(tmp, assetDirEntry.Name()))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(contentBytes)).To(Equal(assetDirEntry.Name()))
			}
		}

		When("all of the requested assets exist in GitHub", func() {
			It("returns all requested assets without error", func() {
				// No additional setup required

				// Run test
				err = src.DownloadReleaseAssets(assets, tmp)

				// Validate results
				Expect(err).ToNot(HaveOccurred())
				checkTmpDir()
			})
		})
		When("at least one of the assets does not exist in GitHub", func() {
			fakeAssetID := int64(987)
			fakeAssetName := "fakeAsset"
			fakeAsset := &gogithub.ReleaseAsset{
				ID: &fakeAssetID,
				Name: &fakeAssetName,
			}
			It("returns an error to handle", func() {
				// No additional setup required

				// Run test
				err = src.DownloadReleaseAssets(append(assets, fakeAsset), tmp)

				// Validate results
				Expect(err).To(HaveOccurred())
			})
			It("downloads assets that do exist", func() {
				// No additional setup required

				// Run test
				err = src.DownloadReleaseAssets(append(assets, fakeAsset), tmp)

				// Validate results
				checkTmpDir()
			})
		})
	})
	When("downloadReleaseAsset() is called and", func() {
		var (
			asset *gogithub.ReleaseAsset
			assetTag = int64(123)
			assetName = "asset1"
			assetContents = "hello world"

			tmp string
		)
		BeforeEach(func() {
			asset = &gogithub.ReleaseAsset{
				Name: &assetName,
				ID: &assetTag,
			}

			tmp, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			err = os.RemoveAll(tmp)
			Expect(err).ToNot(HaveOccurred())
		})
		When("the asset download is redirected", func() {
			It("handles the redirect without error", func() {
				// Setup
				// The following adds a redirect from the normal asset URL pointing to http://<server url>/assetredirect
				redirectPath := "/assetredirect"
				assetRedirectURL, err := url.JoinPath(src.server.server.URL, redirectPath)
				Expect(err).ToNot(HaveOccurred())

				src.server.addDownloadReleaseAssetHandler(src.Owner, src.Repo, asset.GetID(), func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, assetRedirectURL, http.StatusMovedPermanently)
				})
				src.AddResponse(redirectPath, func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprint(w, assetContents)
				})

				// Run test
				err = src.downloadReleaseAsset(asset, tmp)
				
				// Validate results
				Expect(err).ToNot(HaveOccurred())

				testContents, err := ioutil.ReadFile(filepath.Join(tmp, asset.GetName()))
				Expect(err).ToNot(HaveOccurred())
				Expect(assetContents).To(Equal(string(testContents)))
			})
		})
	})
})
