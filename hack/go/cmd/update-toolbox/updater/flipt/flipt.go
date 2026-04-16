// Copyright 2026 Zenauth Ltd.

package flipt

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/semver"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

const digestsAsset = "checksums.txt"

var (
	Tool = updater.Tool{
		Repo: github.Repository{Owner: "flipt-io", Name: "flipt"},
		FindNewerReleaseOptions: []github.FindNewerReleaseOption{
			github.VersionConstraint(func(version semver.Version) bool {
				return semver.Compare(version, "v2") < 0
			}),
		},
		Verify:      verify,
		PostInstall: []string{"flipt", "--version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    "flipt_darwin_arm64.tar.gz",
			Extract: "flipt",
		},
		platform.LinuxARM64: {
			Name:    "flipt_linux_arm64.tar.gz",
			Extract: "flipt",
		},
		platform.LinuxX64: {
			Name:    "flipt_linux_x86_64.tar.gz",
			Extract: "flipt",
		},
	}
)

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	if err := digests.VerifyRelease(release, assets, digestsAsset); err != nil {
		return nil, err
	}

	return updater.DownloadsFromRelease(release, assets)
}
