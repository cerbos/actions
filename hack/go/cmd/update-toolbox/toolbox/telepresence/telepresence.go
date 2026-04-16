// Copyright 2026 Zenauth Ltd.

package telepresence

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
)

var (
	Tool = toolbox.Tool{
		Repo:        github.Repository{Owner: "telepresenceio", Name: "telepresence"},
		Verify:      verify,
		PostInstall: []string{"telepresence", "version"},
	}

	installations = toolbox.Installations{
		platform.DarwinARM64: {Asset: "telepresence-darwin-arm64"},
		platform.LinuxARM64:  {Asset: "telepresence-linux-arm64"},
		platform.LinuxX64:    {Asset: "telepresence-linux-amd64"},
	}
)

func verify(context.Context, *toolbox.Clients, *github.Release) (toolbox.Installations, error) {
	return installations, nil
}
