// Copyright 2026 Zenauth Ltd.

package rmz

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/semver"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
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

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {Name: "aarch64-apple-darwin-rmz"},
		platform.LinuxARM64:  {Name: "aarch64-unknown-linux-gnu-rmz"},
		platform.LinuxX64:    {Name: "x86_64-unknown-linux-gnu-rmz"},
	}
)

func verify(_ context.Context, _ *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	return updater.DownloadsFromRelease(release, assets)
}
