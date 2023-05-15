package ocm

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	gogithub "github.com/google/go-github/v51/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testUtils "github.com/openshift/backplane-tools/pkg/utils/test"
)

// NewTestTool builds an ocm tool within a GithubEnv for testing
func NewTestTool() (*Tool, *testUtils.GithubEnv, error) {
	env, err := testUtils.NewGithubEnv("fake", "test")
	if err != nil {
		return &Tool{}, &testUtils.GithubEnv{}, err
	}
	t := &Tool{
		source: env.Source,
	}
	return t, env, nil
}

// Test_Tool calls the ginkgo specs defined for this package
func Test_Tool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCM Tool")
}

var _ = Describe("Running Install():", func() {
	var (
		// Test objects
		tool *Tool
		env *testUtils.GithubEnv
		err error

		release gogithub.RepositoryRelease
		assets []*gogithub.ReleaseAsset
		binaryAsset *gogithub.ReleaseAsset
		checksumAsset *gogithub.ReleaseAsset
	)

	// Pre-testing setup
	// The following BeforeEach() runs first: initialize any variables needed
	// to run test-specific BeforeEach blocks
	BeforeEach(func() {
		tool, env, err = NewTestTool()
		Expect(err).ToNot(HaveOccurred())
	})
	// The following JustBeforeEach should be the last thing to run prior to testing:
	// initialize any variables that should use the values set in the test-specific
	// BeforeEach blocks
	JustBeforeEach(func() {
			tag := "test-release"
			release = gogithub.RepositoryRelease{
				TagName: &tag,
				Assets: assets,
			}
			err = env.AddFetchLatestReleaseResponse(release)
			Expect(err).ToNot(HaveOccurred())
	})

	// Post-testing cleanup
	AfterEach(func() {
		// Clear the env and asset slice between tests to avoid cross-contamination
		err = env.Cleanup()
		Expect(err).ToNot(HaveOccurred())

		assets = []*gogithub.ReleaseAsset{}
	})

	// Test release download behavior
	When("downloading the release", func() {
		Context("if the latest release can't be downloaded", func() {
			JustBeforeEach(func() {
				// Shutdown test server to simulate GitHub down/unreachable
				env.TestSource.Cleanup()
			})
			It("should return an error for handling", func() {
				// Run test
				err = tool.Install(env.Root, env.Latest)

				// Validate results
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection refused"))
			})
			It("should not modify the install directory", func(){
				// Run test
				err = tool.Install(env.Root, env.Latest)

				// Validate results
				empty, err := env.Empty()
				Expect(err).ToNot(HaveOccurred())
				Expect(empty).To(BeTrue())

				empty, err = env.LatestEmpty()
				Expect(err).ToNot(HaveOccurred())
				Expect(empty).To(BeTrue())
			})
		})
	})
	// Test asset selection behavior
	When("selecting assets and", func() {
		Describe("the binary asset", func() {
			Context("matching this system is not found", func() {
				BeforeEach(func() {
					assets = []*gogithub.ReleaseAsset{
						mismatchedOSBinaryAsset(),
						mismatchedArchBinaryAsset(),
					}
				})
				It("returns an error to handle", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to find a valid ocm binary"))
				})
				It("should not modify the install directory", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					empty, err := env.Empty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())

					empty, err = env.LatestEmpty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())
				})
			})
			Context("has multiple matching entries", func() {
				// TODO
				BeforeEach(func() {
					assets = []*gogithub.ReleaseAsset{
						matchingBinaryAsset(),
						duplicateBinaryAsset(),
					}
				})
				It("should return an error to handle", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("detected duplicate ocm-cli binary assets"))
				})
				It("should not modify the install directory", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					empty, err := env.Empty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())

					empty, err = env.LatestEmpty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())
				})
			})
		})
		Describe("the checksum asset", func() {
			Context("matching this system is not found", func() {
				BeforeEach(func() {
					assets = []*gogithub.ReleaseAsset{
						matchingBinaryAsset(),
						mismatchedOSChecksumAsset(),
						mismatchedArchChecksumAsset(),
					}
				})
				It("returns an error to handle", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to find a valid checksum file"))
				})
				It("should not modify the install directory", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					empty, err := env.Empty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())

					empty, err = env.LatestEmpty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())
				})
			})
			Context("has multiple matching entries", func() {
				BeforeEach(func() {
					assets = []*gogithub.ReleaseAsset{
						matchingChecksumAsset(),
						duplicateChecksumAsset(),
					}
				})
				It("returns an error to handle", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("detected duplicate ocm-cli checksum assets"))
				})
				It("should not modify the install directory", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					empty, err := env.Empty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())

					empty, err = env.LatestEmpty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())
				})
			})
		})
	})
	// Test asset download behavior
	When("downloading assets", func() {
		BeforeEach(func() {
			binaryAsset = matchingBinaryAsset()
			checksumAsset = matchingChecksumAsset()
			assets = []*gogithub.ReleaseAsset{
				binaryAsset,
				checksumAsset,
			}
		})
		Context("fails", func() {
			Context("for the binary asset", func() {
				BeforeEach(func() {
					env.AddDownloadReleaseAssetResponse(checksumAsset, "blahblah")
				})
				It("returns an error to handle", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to download one or more assets"))
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%d", binaryAsset.GetID())))
				})
			})
			Context("for the checksum asset", func() {
				BeforeEach(func() {
					env.AddDownloadReleaseAssetResponse(binaryAsset, "blahblah")
				})
				It("returns an error to handle", func() {
					// Run test
					err = tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to download one or more assets"))
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%d", checksumAsset.GetID())))
				})
			})
		})
		Context("succeeds", func() {
			var (
				// versionedDir refers to the directory within the env's .Root
				// that corresponds to both the correct tool ("ocm" in this case)
				// and the correct version of that tool, which is derived from the
				// release object
				versionedDir string
			)
			BeforeEach(func() {
				// Initialize test variables
				versionedDir = filepath.Join("ocm", release.GetTagName())

				env.AddDownloadReleaseAssetResponse(binaryAsset, "blahblah")
				env.AddDownloadReleaseAssetResponse(checksumAsset, "check")
			})
			It("stores the downloaded assets in the properly versioned directory", func() {
				// Run test
				_ = tool.Install(env.Root, env.Latest)

				// Validate results
				// We should expect an error here since checksumming will fail,
				// so only check the results in the environment
				present, err := env.HasSubdir(versionedDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(present).To(BeTrue())

				// Check binary
				binaryPath := filepath.Join(versionedDir, binaryAsset.GetName())
				binaryFile, err := env.Open(binaryPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(binaryFile).ToNot(BeNil())
				defer func() {
					err = binaryFile.Close()
					Expect(err).ToNot(HaveOccurred())
				}()

				contents, err := io.ReadAll(binaryFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(Equal("blahblah"))

				// Check checksum
				checksumPath := filepath.Join(versionedDir, checksumAsset.GetName())
				checksumFile, err := env.Open(checksumPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(checksumFile).ToNot(BeNil())
				defer func() {
					err = checksumFile.Close()
					Expect(err).ToNot(HaveOccurred())
				}()

				contents, err = io.ReadAll(checksumFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(Equal("check"))
			})
			It("saves the binary asset as an executable file", func() {
				// Run test
				_ = tool.Install(env.Root, env.Latest)

				// Validate results
				// We should expect an error here since checksumming will fail,
				// so only check the results in the environment
				present, err := env.HasSubdir(versionedDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(present).To(BeTrue())

				binaryPath := filepath.Join(versionedDir, binaryAsset.GetName())
				binaryFile, err := env.Open(binaryPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(binaryFile).ToNot(BeNil())
				defer func() {
					err = binaryFile.Close()
					Expect(err).ToNot(HaveOccurred())
				}()
				info, err := binaryFile.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(info.Mode()).To(Equal(os.FileMode(0755)))
			})
			It("does not modify previous release installs", func() {
				// Setup
				// Create a faux previous-version directory and populate it
				fakeDir := filepath.Join("ocm", "previous-release")
				err := env.MkdirAll(fakeDir, os.FileMode(0755))
				Expect(err).ToNot(HaveOccurred())

				fakeBin := filepath.Join(fakeDir, "fake-bin")
				fakeBinMode := os.FileMode(0755)
				fakeBinContents := "blahblah"
				_, err = env.Create(fakeBin, fakeBinMode, fakeBinContents)
				Expect(err).ToNot(HaveOccurred())

				fakeCheck := filepath.Join(fakeDir, "fake-checksum")
				fakeCheckMode := os.FileMode(0644)
				fakeCheckContents := "checksum"
				_, err = env.Create(fakeCheck, fakeCheckMode, fakeCheckContents)
				Expect(err).ToNot(HaveOccurred())

				// Run test
				_ = tool.Install(env.Root, env.Latest)

				// Validate results

				// Check the faux dir
				dir, err := env.Open(fakeDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(dir).ToNot(BeNil())
				defer func() {
					err = dir.Close()
					Expect(err).ToNot(HaveOccurred())
				}()
				dirInfo, err := dir.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(dirInfo.IsDir()).To(BeTrue())

				// Check the faux binary
				binFile, err := env.Open(fakeBin)
				Expect(err).ToNot(HaveOccurred())
				Expect(binFile).ToNot(BeNil())
				defer func() {
					err = binFile.Close()
					Expect(err).ToNot(HaveOccurred())
				}()
				contents, err := io.ReadAll(binFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(Equal(fakeBinContents))
				binInfo, err := binFile.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(binInfo.Mode()).To(Equal(fakeBinMode))

				// Check the faux checksum
				checkFile, err := env.Open(fakeCheck)
				Expect(err).ToNot(HaveOccurred())
				Expect(checkFile).ToNot(BeNil())
				defer func() {
					err = checkFile.Close()
					Expect(err).ToNot(HaveOccurred())
				}()
				contents, err = io.ReadAll(checkFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(Equal(fakeCheckContents))
				checkInfo, err := checkFile.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(checkInfo.Mode()).To(Equal(fakeCheckMode))
			})
		})
	})
	// Test checksum behavior
	When("the checksum calculations", func() {
		var (
			binaryAssetContents string
		)
		BeforeEach(func() {
			binaryAsset = matchingBinaryAsset()
			checksumAsset = matchingChecksumAsset()
			assets = []*gogithub.ReleaseAsset{
				binaryAsset,
				checksumAsset,
			}

			binaryAssetContents = "blahblah"
			env.AddDownloadReleaseAssetResponse(binaryAsset, binaryAssetContents)
		})
		Context("fail", func() {
			Context("because the file has an invalid format", func() {
				BeforeEach(func() {
					env.AddDownloadReleaseAssetResponse(checksumAsset, "invalidchecksumformat")
				})
				It("returns an error to handle", func() {
					// Run test
					err := tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("invalid checksum file: expected 2 tokens"))
				})
				It("does not link the downloaded binary asset to the latest directory", func(){
					// Run test
					_ = tool.Install(env.Root, env.Latest)

					// Validate results
					empty, err := env.LatestEmpty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())
				})
			})
			Context("because the checksum is incorrect", func() {
				BeforeEach(func() {
					env.AddDownloadReleaseAssetResponse(checksumAsset, "invalidchecksumvalue checksumfile")
				})
				It("returns an error to handle", func() {
					// Run test
					err := tool.Install(env.Root, env.Latest)

					// Validate results
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("the provided checksum for ocm-cli"))
				})
				It("does not link the downloaded binary asset to the latest directory", func(){
					// Run test
					_ = tool.Install(env.Root, env.Latest)

					// Validate results
					empty, err := env.LatestEmpty()
					Expect(err).ToNot(HaveOccurred())
					Expect(empty).To(BeTrue())
				})
			})
		})
		Context("succeed", func() {
			BeforeEach(func() {
				sumBytes := sha256.Sum256([]byte(binaryAssetContents))
				checksumFile := fmt.Sprintf("%x filename", sumBytes[:])
				env.AddDownloadReleaseAssetResponse(checksumAsset, checksumFile)
			})
			It("does not return an error", func() {
				// Run test
				err := tool.Install(env.Root, env.Latest)

				// Validate results
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
	When("linking the binary asset", func() {
		Context("if a link is already present", func() {
			BeforeEach(func() {
				_, _, _, err = generatePreviousInstall(env, "oldblahblah", "oldcheck")

				binaryAsset = matchingBinaryAsset()
				checksumAsset = matchingChecksumAsset()
				assets = []*gogithub.ReleaseAsset{
					binaryAsset,
					checksumAsset,
				}

				binaryAssetContents := "blahblah"
				env.AddDownloadReleaseAssetResponse(binaryAsset, binaryAssetContents)
				sumBytes := sha256.Sum256([]byte(binaryAssetContents))
				checksumFileContents := fmt.Sprintf("%x filename", sumBytes[:])
				env.AddDownloadReleaseAssetResponse(checksumAsset, checksumFileContents)
			})
			It("replaces it and links to the new binary", func() {
				// Run test
				err := tool.Install(env.Root, env.Latest)

				// Validate results
				Expect(err).ToNot(HaveOccurred())
				valid, err := env.IsLink("latest/ocm")
				Expect(err).ToNot(HaveOccurred())
				Expect(valid).To(BeTrue())

				linkedTo, err := env.EvalSymlink("latest/ocm")
				Expect(err).ToNot(HaveOccurred())
				// The absolute path expected is the tool dir (root dir + "ocm/") + release tag ("test-release") + binary name (the asset's name)
				expectedToolPath := filepath.Join(tool.toolDir(env.Root), release.GetTagName(), binaryAsset.GetName())
				Expect(linkedTo).To(Equal(expectedToolPath))
			})
		})
	})
})

var _ = Describe("Running Remove():", func() {
	var (
		// Test objects
		tool *Tool
		env *testUtils.GithubEnv
		err error
	)
	BeforeEach(func() {
		tool, env, err = NewTestTool()
		Expect(err).ToNot(HaveOccurred())

		_, _, _, err = generatePreviousInstall(env, "blahblah", "check")
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		err = env.Cleanup()
		Expect(err).ToNot(HaveOccurred())
	})
	When("removing the tool directory", func() {
		Context("and there is nothing to remove", func() {
			BeforeEach(func() {
				err = env.RemoveAll(tool.toolDir(env.Root))
				Expect(err).ToNot(HaveOccurred())
			})
			It("it does not return an error", func() {
				// Run test
				err = tool.Remove(env.Root, env.Latest)

				// Validate results
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("and there are other tools installed", func() {
			const fauxToolContents string = "blahblah"
			var (
				fauxTool *os.File
				fauxInfo os.FileInfo
			)
			BeforeEach(func() {
				fauxToolDir := filepath.Join(env.Root, "faux")
				err = env.MkdirAll(fauxToolDir, os.FileMode(0755))
				Expect(err).ToNot(HaveOccurred())

				fauxToolPath := filepath.Join(fauxToolDir, "tool")
				fauxTool, err = env.Create(fauxToolPath, os.FileMode(0755), fauxToolContents)
				Expect(err).ToNot(HaveOccurred())
				Expect(fauxTool).ToNot(BeNil())
				fauxInfo, err = fauxTool.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fauxInfo).ToNot(BeNil())
			})
			AfterEach(func() {
				err = fauxTool.Close()
				Expect(err).ToNot(HaveOccurred())
			})
			It("does not modify them", func() {
				// Run test
				err = tool.Remove(env.Root, env.Latest)

				// Validate results
				Expect(err).ToNot(HaveOccurred())

				currentInfo, err := fauxTool.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(currentInfo).To(Equal(fauxInfo))

				currentContents, err := io.ReadAll(fauxTool)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(currentContents)).To(Equal(fauxToolContents))
			})
		})
	})
	When("removing the symlinked binary", func() {
		Context("and there is no symlink to remove", func() {
			BeforeEach(func() {
				err = env.RemoveAll("latest/ocm")
				Expect(err).ToNot(HaveOccurred())
			})
			It("does not return an error", func() {
				// Run test
				err = tool.Remove(env.Root, env.Latest)

				// Validate results
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("and there are other files present in the latest directory",func() {
			const fileContents string = "blahblah"
			var (
				file *os.File
				originalInfo os.FileInfo
			)
			BeforeEach(func() {
				file, err = env.Create("latest/other-file", os.FileMode(0755), fileContents)
				Expect(err).ToNot(HaveOccurred())
				Expect(file).ToNot(BeNil())

				originalInfo, err = file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(originalInfo).ToNot(BeNil())
			})
			AfterEach(func() {
				err = file.Close()
				Expect(err).ToNot(HaveOccurred())
			})
			It("does not modify the other files", func() {
				// Run test
				err = tool.Remove(env.Root, env.Latest)

				// Validate results
				Expect(err).ToNot(HaveOccurred())

				currentInfo, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(currentInfo).To(Equal(originalInfo))

				currentContents, err := io.ReadAll(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(currentContents)).To(Equal(fileContents))
			})
		})
	})
})

// matchingChecksumAsset builds a checksum asset whose name matches the 
// current system's OS+arch
func matchingChecksumAsset() *gogithub.ReleaseAsset {
	id := int64(987)
	name := fmt.Sprintf("ocm-%s-%s.sha256", runtime.GOOS, runtime.GOARCH)
	asset := &gogithub.ReleaseAsset {
		ID: &id,
		Name: &name,
	}
	return asset
}

// duplicateChecksumAsset creates a checksum asset with the same name as the
// one generated by matchingChecksumAsset(), but with a different ID
func duplicateChecksumAsset() *gogithub.ReleaseAsset {
	asset := matchingChecksumAsset()
	id := int64(789)
	asset.ID = &id
	return asset
}

// mismatchedOSChecksumAsset builds a checksum asset whose name matches the
// current system's arch, but not OS
func mismatchedOSChecksumAsset() *gogithub.ReleaseAsset {
	id := int64(876)
	name := fmt.Sprintf("ocm-incorrectOS-%s.sha256", runtime.GOARCH)
	asset := &gogithub.ReleaseAsset{
		ID: &id,
		Name: &name,
	}
	return asset
}

// mismatchedArchChecksumAsset builds a checksum asset whose name matches the
// current system's OS, but not architecture
func mismatchedArchChecksumAsset() *gogithub.ReleaseAsset {
	id := int64(765)
	name := fmt.Sprintf("ocm-%s-incorrectArch.sha256", runtime.GOOS)
	asset := &gogithub.ReleaseAsset{
		ID: &id,
		Name: &name,
	}
	return asset
}

// matchingBinaryAsset builds a binary asset whose name matches the current
// system's OS+arch
func matchingBinaryAsset() *gogithub.ReleaseAsset {
	id := int64(123)
	name := fmt.Sprintf("ocm-%s-%s", runtime.GOOS, runtime.GOARCH)
	asset := &gogithub.ReleaseAsset{
		ID: &id,
		Name: &name,
	}
	return asset
}

// duplicateBinaryAsset creates a binary asset with the same name as the
// one generated by matchingBinaryAsset(), but with a different ID
func duplicateBinaryAsset() *gogithub.ReleaseAsset {
	asset := matchingBinaryAsset()
	id := int64(321)
	asset.ID = &id
	return asset
}

// mismatchedOSBinaryAsset builds a binary asset whose name matches the current
// system's arch, but not OS
func mismatchedOSBinaryAsset() *gogithub.ReleaseAsset {
	id := int64(234)
	name := fmt.Sprintf("ocm-incorrectOS-%s", runtime.GOARCH)
	asset := &gogithub.ReleaseAsset{
		ID: &id,
		Name: &name,
	}
	return asset
}

// mismatchedArchBinaryAsset builds a binary asset whose name matches the current
// system's OS, but not architecture
func mismatchedArchBinaryAsset() *gogithub.ReleaseAsset {
	id := int64(345)
	name := fmt.Sprintf("ocm-%s-incorrectArch", runtime.GOOS)
	asset := &gogithub.ReleaseAsset{
		ID: &id,
		Name: &name,
	}
	return asset
}

func generatePreviousInstall(env *testUtils.GithubEnv, binContents string, checkContents string) (dir string, binary *os.File, checksum *os.File, err error) {
	dir = filepath.Join("ocm", "previous-release")
	err = env.MkdirAll(dir, os.FileMode(0755))
	if err != nil {
		return
	}

	binPath := filepath.Join(dir, "fake-bin")
	binMode := os.FileMode(0755)
	binary, err = env.Create(binPath, binMode, binContents)
	if err != nil {
		return
	}
	err = env.CreateLink(binPath, "latest/ocm")
	if err != nil {
		return
	}

	checkPath := filepath.Join(dir, "fake-checksum")
	checkMode := os.FileMode(0644)
	checksum, err = env.Create(checkPath, checkMode, checkContents)
	return
}
