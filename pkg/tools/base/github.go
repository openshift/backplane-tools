package base

import (
	"github.com/openshift/backplane-tools/pkg/sources/github"
)

type Github struct {
	Default
	Source             *github.Source
	VersionInLatestTag bool
}

func (t *Github) _LatestVersion() (string, error) {
	if t.VersionInLatestTag {
		return t.Source.FetchLatestTag()
	}
	release, err := t.Source.FetchLatestRelease()
	if err != nil {
		return "", err
	}
	return release.GetTagName(), nil
}

func (t *Github) LatestVersion() (string, error) {
	if t.latestVersion == "" {
		version, err := t._LatestVersion()
		if err != nil {
			return "", err
		}
		t.latestVersion = version
	}
	return t.latestVersion, nil
}
