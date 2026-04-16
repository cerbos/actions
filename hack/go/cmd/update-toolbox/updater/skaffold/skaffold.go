// Copyright 2026 Zenauth Ltd.

package skaffold

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

// https://github.com/GoogleContainerTools/skaffold/blob/c186fff81c8031cec0927df89aecd52ce6623eb0/KEYS
const publicKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWZrGCUaJJr1H8a36sG4UUoXvlXvZ
wQfk16sxprI2gOJ2vFFggdq3ixF2h4qNBt0kI7ciDhgpwS8t+/960IsIgw==
-----END PUBLIC KEY-----`

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "GoogleContainerTools", Name: "skaffold"},
		Verify:      verify,
		PostInstall: []string{"skaffold", "version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {Name: "skaffold-darwin-arm64"},
		platform.LinuxARM64:  {Name: "skaffold-linux-arm64"},
		platform.LinuxX64:    {Name: "skaffold-linux-amd64"},
	}
)

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	ecdsa, err := signing.NewECDSA(publicKey)
	if err != nil {
		return nil, err
	}

	const downloadsPerAsset = 2
	downloadAssets := make([]string, 0, downloadsPerAsset*len(assets))
	for _, asset := range assets {
		downloadAssets = append(downloadAssets, digestAssetName(asset), signatureAssetName(asset))
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, downloadAssets...); err != nil {
		return nil, err
	}

	for _, asset := range assets {
		if err := verifyAsset(ecdsa, release, asset); err != nil {
			return nil, fmt.Errorf("failed to verify %s: %w", asset.Name, err)
		}
	}

	return updater.DownloadsFromRelease(release, assets)
}

func verifyAsset(ecdsa *signing.ECDSA, release *github.Release, asset updater.AssetToDownload) error {
	digestAsset, err := release.Asset(digestAssetName(asset))
	if err != nil {
		return err
	}

	signatureAsset, err := release.Asset(signatureAssetName(asset))
	if err != nil {
		return err
	}

	if err := ecdsa.Verify(digestAsset.Digest, signatureAsset.Contents); err != nil {
		return err
	}

	digest, err := digests.FromAsset(digestAsset)
	if err != nil {
		return err
	}

	return digests.VerifyAsset(release, asset, digest)
}

func digestAssetName(asset updater.AssetToDownload) string {
	return asset.Name + ".sha256"
}

func signatureAssetName(asset updater.AssetToDownload) string {
	return digestAssetName(asset) + ".sig"
}
