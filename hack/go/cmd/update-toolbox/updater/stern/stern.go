// Copyright 2026 Zenauth Ltd.

package stern

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

const digestsAsset = "checksums.txt"

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "stern", Name: "stern"},
	Verify:      verify,
	PostInstall: []string{"stern", "--version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	version := release.Version.Number()

	assets := updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    fmt.Sprintf("stern_%s_darwin_arm64.tar.gz", version),
			Extract: "stern",
		},
		platform.LinuxARM64: {
			Name:    fmt.Sprintf("stern_%s_linux_arm64.tar.gz", version),
			Extract: "stern",
		},
		platform.LinuxX64: {
			Name:    fmt.Sprintf("stern_%s_linux_amd64.tar.gz", version),
			Extract: "stern",
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
