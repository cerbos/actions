// Copyright 2026 Zenauth Ltd.

package golangcilint

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "golangci", Name: "golangci-lint"},
	Verify:      verify,
	PostInstall: []string{"golangci-lint", "version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	version := release.Version.Number()

	assets := updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    fmt.Sprintf("golangci-lint-%s-darwin-arm64.tar.gz", version),
			Extract: fmt.Sprintf("golangci-lint-%s-darwin-arm64/golangci-lint", version),
		},
		platform.LinuxARM64: {
			Name:    fmt.Sprintf("golangci-lint-%s-linux-arm64.tar.gz", version),
			Extract: fmt.Sprintf("golangci-lint-%s-linux-arm64/golangci-lint", version),
		},
		platform.LinuxX64: {
			Name:    fmt.Sprintf("golangci-lint-%s-linux-amd64.tar.gz", version),
			Extract: fmt.Sprintf("golangci-lint-%s-linux-amd64/golangci-lint", version),
		},
	}

	digestsAsset := fmt.Sprintf("golangci-lint-%s-checksums.txt", version)

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	if err := digests.VerifyRelease(release, assets, digestsAsset); err != nil {
		return nil, err
	}

	return updater.DownloadsFromRelease(release, assets)
}
