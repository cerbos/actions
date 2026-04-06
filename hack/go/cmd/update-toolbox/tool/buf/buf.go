// Copyright 2021-2026 Zenauth Ltd.

package buf

import (
	"context"
	"fmt"

	"aead.dev/minisign"

	"github.com/cerbos/actions/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/cmd/update-toolbox/tool"
	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/semver"
)

const (
	digestsAsset   = "sha256.txt"
	signatureAsset = "sha256.txt.minisig"
)

var (
	binaries = map[tool.Platform]string{
		tool.LinuxARM64: "buf-Linux-aarch64",
		tool.LinuxX64:   "buf-Linux-x86_64",
	}

	publicKey minisign.PublicKey
)

func init() {
	// https://buf.build/docs/cli/installation/#verifying-a-release
	if err := publicKey.UnmarshalText([]byte("RWQ/i9xseZwBVE7pEniCNjlNOeeyp4BQgdZDLQcAohxEAH5Uj5DEKjv6")); err != nil {
		panic(err)
	}
}

func Update(ctx context.Context, client *github.Client, oldVersion semver.Version) (*tool.Source, error) {
	release, err := client.FindNewerRelease(ctx, github.Repository{Owner: "bufbuild", Name: "buf"}, oldVersion)
	if err != nil {
		return nil, err
	}

	if release == nil {
		return nil, nil
	}

	source, err := verify(ctx, client, release)
	if err != nil {
		return nil, fmt.Errorf("failed to verify %s: %w", release, err)
	}

	return source, nil
}

func verify(ctx context.Context, client *github.Client, release *github.Release) (*tool.Source, error) {
	if err := client.DownloadAssets(ctx, release, digestsAsset, signatureAsset); err != nil {
		return nil, err
	}

	digestsFile := release.Assets[digestsAsset].Contents

	if !minisign.Verify(publicKey, digestsFile, release.Assets[signatureAsset].Contents) {
		return nil, fmt.Errorf("signature verification failed for %s", release)
	}

	digests, err := digests.Parse(digestsFile)
	if err != nil {
		return nil, err
	}

	source := &tool.Source{
		Version:     release.Version,
		Downloads:   make(map[tool.Platform]tool.Download, len(binaries)),
		PostInstall: []string{"buf", "--version"},
	}

	for platform, binary := range binaries {
		digest, ok := digests[binary]
		if !ok {
			return nil, fmt.Errorf("missing digest for %s", binary)
		}

		asset, err := release.Asset(binary)
		if err != nil {
			return nil, err
		}

		if digest != asset.Digest {
			return nil, fmt.Errorf("digest mismatch for %s", binary)
		}

		source.Downloads[platform] = tool.Download{
			URL:    asset.URL,
			Digest: asset.Digest,
		}
	}

	return source, nil
}
