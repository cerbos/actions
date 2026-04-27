// Copyright 2026 Zenauth Ltd.

package kind

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "kubernetes-sigs", Name: "kind"},
		Verify:      verify,
		PostInstall: []string{"kind", "version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {Name: "kind-darwin-arm64"},
		platform.LinuxARM64:  {Name: "kind-linux-arm64"},
		platform.LinuxX64:    {Name: "kind-linux-amd64"},
	}
)

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	digestAssets := make([]string, 0, len(assets))
	for _, asset := range assets {
		digestAssets = append(digestAssets, digestAssetName(asset))
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, digestAssets...); err != nil {
		return nil, err
	}

	for _, asset := range assets {
		if err := verifyAsset(release, asset); err != nil {
			return nil, fmt.Errorf("failed to verify %s: %w", asset.Name, err)
		}
	}

	return updater.DownloadsFromRelease(release, assets)
}

func verifyAsset(release *github.Release, asset updater.AssetToDownload) error {
	digest, err := digests.FromRelease(release, digestAssetName(asset))
	if err != nil {
		return err
	}

	return digests.VerifyAsset(release, asset, digest)
}

func digestAssetName(asset updater.AssetToDownload) string {
	return asset.Name + ".sha256sum"
}
