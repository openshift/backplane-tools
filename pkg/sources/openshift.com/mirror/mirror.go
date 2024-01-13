/*
mirror provides the capability for tools to retrieve files from mirror.openshift.com
*/
package mirror

import (
	"github.com/openshift/backplane-tools/pkg/sources/base/url"
)

const (
	baseURL string = "http://mirror.openshift.com"
)

// Source objects retrieve files from a mirror server
type Source struct {
	// baseURL represents the url that the Source's requests should be built off of
	*url.Source
}

// NewSource creates a Source
func NewSource() *Source {
	s := &Source{
		Source: url.NewSource(baseURL),
	}
	return s
}
