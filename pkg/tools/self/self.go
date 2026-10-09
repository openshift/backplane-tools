package self

import (
	"github.com/openshift/backplane-tools/pkg/tools/base"
)

type Tool struct {
	base.GoBin
}

func New() *Tool {
	return &Tool{
		GoBin: base.GoBin{
			Default: base.NewDefault("backplane-tools"),
			Module:  "github.com/openshift/backplane-tools",
			Branch:  "main",
		},
	}
}
