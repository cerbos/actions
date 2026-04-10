// Copyright 2026 Zenauth Ltd.

package rmz

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/semver"
)

var (
	Tool = toolbox.Tool{
		Repo: github.Repository{Owner: "SUPERCILEX", Name: "fuc"},
		FindNewerReleaseOptions: []github.FindNewerReleaseOption{
			github.VersionFromTag(func(tag string) semver.Version {
				return semver.Version("v" + tag)
			}),
		},
		Verify:      verify,
		PostInstall: []string{"rmz", "--version"},
	}

	installations = toolbox.Installations{
		toolbox.LinuxARM64: {Asset: "aarch64-unknown-linux-gnu-rmz"},
		toolbox.LinuxX64:   {Asset: "x86_64-unknown-linux-gnu-rmz"},
	}
)

func verify(context.Context, *toolbox.Clients, *github.Release) (toolbox.Installations, error) {
	return installations, nil
}
