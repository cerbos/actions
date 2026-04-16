// Copyright 2026 Zenauth Ltd.

package buf

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/signing"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

const (
	digestsAsset   = "sha256.txt"
	signatureAsset = "sha256.txt.minisig"

	// https://buf.build/docs/cli/installation/#verifying-a-release
	publicKey = "RWQ/i9xseZwBVE7pEniCNjlNOeeyp4BQgdZDLQcAohxEAH5Uj5DEKjv6"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "bufbuild", Name: "buf"},
		Verify:      verify,
		PostInstall: []string{"buf", "--version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {Name: "buf-Darwin-arm64"},
		platform.LinuxARM64:  {Name: "buf-Linux-aarch64"},
		platform.LinuxX64:    {Name: "buf-Linux-x86_64"},
	}
)

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	minisign, err := signing.NewMinisign(publicKey)
	if err != nil {
		return nil, err
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset, signatureAsset); err != nil {
		return nil, err
	}

	if err := minisign.Verify(release.Assets[digestsAsset].Contents, release.Assets[signatureAsset].Contents); err != nil {
		return nil, err
	}

	if err := digests.VerifyRelease(release, assets, digestsAsset); err != nil {
		return nil, err
	}

	return updater.DownloadsFromRelease(release, assets)
}
