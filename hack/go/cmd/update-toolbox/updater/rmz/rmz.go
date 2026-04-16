// Copyright 2026 Zenauth Ltd.

package rmz

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/semver"
)

var (
	Tool = updater.Tool{
		Repo: github.Repository{Owner: "SUPERCILEX", Name: "fuc"},
		FindNewerReleaseOptions: []github.FindNewerReleaseOption{
			github.VersionFromTag(func(tag string) semver.Version {
				return semver.Version("v" + tag)
			}),
		},
		Verify:      verify,
		PostInstall: []string{"rmz", "--version"},
	}

	installations = updater.Installations{
		platform.DarwinARM64: {Asset: "aarch64-apple-darwin-rmz"},
		platform.LinuxARM64:  {Asset: "aarch64-unknown-linux-gnu-rmz"},
		platform.LinuxX64:    {Asset: "x86_64-unknown-linux-gnu-rmz"},
	}
)

func verify(context.Context, *updater.Clients, *github.Release) (updater.Installations, error) {
	return installations, nil
}
