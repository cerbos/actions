// Copyright 2026 Zenauth Ltd.

package oras

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/signing"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "oras-project", Name: "oras"},
	Verify:      verify,
	PostInstall: []string{"oras", "version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	pgp, err := signing.NewPGP(publicKeys)
	if err != nil {
		return nil, err
	}

	version := release.Version.Number()

	assets := updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    fmt.Sprintf("oras_%s_darwin_arm64.tar.gz", version),
			Extract: "oras",
		},
		platform.LinuxARM64: {
			Name:    fmt.Sprintf("oras_%s_linux_arm64.tar.gz", version),
			Extract: "oras",
		},
		platform.LinuxX64: {
			Name:    fmt.Sprintf("oras_%s_linux_amd64.tar.gz", version),
			Extract: "oras",
		},
	}

	digestsAsset := fmt.Sprintf("oras_%s_checksums.txt", version)
	signatureAsset := digestsAsset + ".asc"

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset, signatureAsset); err != nil {
		return nil, err
	}

	if err := pgp.Verify(release.Assets[digestsAsset].Contents, release.Assets[signatureAsset].Contents); err != nil {
		return nil, err
	}

	if err := digests.VerifyRelease(release, assets, digestsAsset); err != nil {
		return nil, err
	}

	return updater.DownloadsFromRelease(release, assets)
}
