// Copyright 2026 Zenauth Ltd.

package digests

import (
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/github"
)

func FromRelease(release *github.Release, assetName string) (digest.Digests, error) {
	asset, err := release.Asset(assetName)
	if err != nil {
		return nil, err
	}

	return FromAsset(asset)
}

func FromAsset(asset *github.Asset) (digest.Digests, error) {
	return digest.ParseFile(asset.Contents)
}

func VerifyRelease(release *github.Release, assets updater.AssetsToDownload, digestsAssetName string) error {
	digests, err := FromRelease(release, digestsAssetName)
	if err != nil {
		return err
	}

	return VerifyAssets(release, assets, digests)
}

func VerifyAssets(release *github.Release, assets updater.AssetsToDownload, digests digest.Digests) error {
	for _, asset := range assets {
		if err := VerifyAsset(release, asset, digests); err != nil {
			return err
		}
	}

	return nil
}

func VerifyAsset(release *github.Release, assetToDownload updater.AssetToDownload, digests digest.Digests) error {
	asset, err := release.Asset(assetToDownload.Name)
	if err != nil {
		return err
	}

	digest, ok := digests[asset.Name]
	if !ok {
		return fmt.Errorf("missing digest for %s", asset.Name)
	}

	if digest != asset.Digest {
		return fmt.Errorf("digest mismatch for %s", asset.Name)
	}

	return nil
}
