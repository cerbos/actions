// Copyright 2021-2026 Zenauth Ltd.

package flipt

import (
	"context"

	"github.com/cerbos/actions/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/semver"
)

const digestsAsset = "checksums.txt"

var (
	Tool = toolbox.Tool{
		Repo: github.Repository{Owner: "flipt-io", Name: "flipt"},
		FindNewerReleaseOptions: []github.FindNewerReleaseOption{
			github.VersionConstraint(func(version semver.Version) bool {
				return semver.Compare(version, "v2") < 0
			}),
		},
		Verify:      verify,
		PostInstall: []string{"flipt", "--version"},
	}

	installations = toolbox.Installations{
		toolbox.LinuxARM64: {
			Asset:   "flipt_linux_arm64.tar.gz",
			Extract: "flipt",
		},
		toolbox.LinuxX64: {
			Asset:   "flipt_linux_x86_64.tar.gz",
			Extract: "flipt",
		},
	}
)

func verify(ctx context.Context, client *github.Client, release *github.Release) (toolbox.Installations, error) {
	if err := client.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
