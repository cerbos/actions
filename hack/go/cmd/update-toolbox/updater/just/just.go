// Copyright 2026 Zenauth Ltd.

package just

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/semver"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

const digestsAsset = "SHA256SUMS"

var Tool = updater.Tool{
	Repo: github.Repository{Owner: "casey", Name: "just"},
	FindNewerReleaseOptions: []github.FindNewerReleaseOption{
		github.VersionFromTag(func(tag string) semver.Version {
			return semver.Version("v" + tag)
		}),
	},
	Verify:      verify,
	PostInstall: []string{"just", "--version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	version := release.Version.Number()

	assets := updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    fmt.Sprintf("just-%s-aarch64-apple-darwin.tar.gz", version),
			Extract: "just",
		},
		platform.LinuxARM64: {
			Name:    fmt.Sprintf("just-%s-aarch64-unknown-linux-musl.tar.gz", version),
			Extract: "just",
		},
		platform.LinuxX64: {
			Name:    fmt.Sprintf("just-%s-x86_64-unknown-linux-musl.tar.gz", version),
			Extract: "just",
		},
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	if err := digests.VerifyRelease(release, assets, digestsAsset); err != nil {
		return nil, err
	}

	return updater.DownloadsFromRelease(release, assets)
}
