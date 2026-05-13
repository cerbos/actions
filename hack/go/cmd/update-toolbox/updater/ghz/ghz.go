// Copyright 2026 Zenauth Ltd.

package ghz

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "bojand", Name: "ghz"},
		Verify:      verify,
		PostInstall: []string{"ghz", "--version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    "ghz-darwin-arm64.tar.gz",
			Extract: "ghz",
		},
		platform.LinuxARM64: {
			Name:    "ghz-linux-arm64.tar.gz",
			Extract: "ghz",
		},
		platform.LinuxX64: {
			Name:    "ghz-linux-x86_64.tar.gz",
			Extract: "ghz",
		},
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

func verifyAsset(release *github.Release, assetToDownload updater.AssetToDownload) error {
	asset, err := release.Asset(assetToDownload.Name)
	if err != nil {
		return err
	}

	digestAsset, err := release.Asset(digestAssetName(assetToDownload))
	if err != nil {
		return err
	}

	assetDigest, err := digest.Parse(string(bytes.TrimSpace(digestAsset.Contents)))
	if err != nil {
		return err
	}

	if assetDigest != asset.Digest {
		return digest.ErrMismatch
	}

	return nil
}

func digestAssetName(asset updater.AssetToDownload) string {
	return asset.Name + ".sha256"
}
