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

func Verify(release *github.Release, installations updater.Installations, digestsAssetName string) error {
	digests, err := FromRelease(release, digestsAssetName)
	if err != nil {
		return err
	}

	return VerifyInstallations(release, installations, digests)
}

func VerifyInstallations(release *github.Release, installations updater.Installations, digests digest.Digests) error {
	for _, installation := range installations {
		if err := VerifyInstallation(release, installation, digests); err != nil {
			return err
		}
	}

	return nil
}

func VerifyInstallation(release *github.Release, installation updater.Installation, digests digest.Digests) error {
	asset, err := release.Asset(installation.Asset)
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
