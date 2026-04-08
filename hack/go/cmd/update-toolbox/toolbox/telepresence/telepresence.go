// Copyright 2021-2026 Zenauth Ltd.

package telepresence

import (
	"context"

	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/internal/github"
)

var (
	Tool = toolbox.Tool{
		Repo:        github.Repository{Owner: "telepresenceio", Name: "telepresence"},
		Verify:      verify,
		PostInstall: []string{"telepresence", "version"},
	}

	installations = toolbox.Installations{
		toolbox.LinuxARM64: {Asset: "telepresence-linux-arm64"},
		toolbox.LinuxX64:   {Asset: "telepresence-linux-amd64"},
	}
)

func verify(context.Context, *toolbox.Clients, *github.Release) (toolbox.Installations, error) {
	return installations, nil
}
