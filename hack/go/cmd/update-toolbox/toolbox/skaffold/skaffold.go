// Copyright 2026 Zenauth Ltd.

package skaffold

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
)

var (
	Tool = toolbox.Tool{
		Repo:        github.Repository{Owner: "GoogleContainerTools", Name: "skaffold"},
		Verify:      verify,
		PostInstall: []string{"skaffold", "version"},
	}

	installations = toolbox.Installations{
		platform.DarwinARM64: {Asset: "skaffold-darwin-arm64"},
		platform.LinuxARM64:  {Asset: "skaffold-linux-arm64"},
		platform.LinuxX64:    {Asset: "skaffold-linux-amd64"},
	}

	publicKey *ecdsa.PublicKey
)

func init() {
	// https://github.com/GoogleContainerTools/skaffold/blob/c186fff81c8031cec0927df89aecd52ce6623eb0/KEYS
	block, _ := pem.Decode([]byte(`-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWZrGCUaJJr1H8a36sG4UUoXvlXvZ
wQfk16sxprI2gOJ2vFFggdq3ixF2h4qNBt0kI7ciDhgpwS8t+/960IsIgw==
-----END PUBLIC KEY-----`))
	if block == nil {
		panic("failed to parse PEM-encoded public key")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic("failed to parse DER-encoded public key")
	}

	var ok bool
	publicKey, ok = key.(*ecdsa.PublicKey)
	if !ok {
		panic(fmt.Errorf("expected ECDSA public key, got %T", key))
	}
}

func verify(ctx context.Context, clients *toolbox.Clients, release *github.Release) (toolbox.Installations, error) {
	const downloadsPerInstallation = 2
	downloadAssets := make([]string, 0, downloadsPerInstallation*len(installations))
	for _, installation := range installations {
		downloadAssets = append(downloadAssets, digestAssetName(installation), signatureAssetName(installation))
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, downloadAssets...); err != nil {
		return nil, err
	}

	for _, installation := range installations {
		if err := verifyInstallation(release, installation); err != nil {
			return nil, fmt.Errorf("failed to verify %s: %w", installation.Asset, err)
		}
	}

	return installations, nil
}

func verifyInstallation(release *github.Release, installation toolbox.Installation) error {
	digestAsset, err := release.Asset(digestAssetName(installation))
	if err != nil {
		return err
	}

	signatureAsset, err := release.Asset(signatureAssetName(installation))
	if err != nil {
		return err
	}

	if !ecdsa.VerifyASN1(publicKey, digestAsset.Digest[:], signatureAsset.Contents) {
		return errors.New("invalid signature")
	}

	digest, err := digests.FromAsset(digestAsset)
	if err != nil {
		return err
	}

	return digests.VerifyInstallation(release, installation, digest)
}

func digestAssetName(installation toolbox.Installation) string {
	return installation.Asset + ".sha256"
}

func signatureAssetName(installation toolbox.Installation) string {
	return digestAssetName(installation) + ".sig"
}
