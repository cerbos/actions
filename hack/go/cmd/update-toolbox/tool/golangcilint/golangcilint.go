// Copyright 2021-2026 Zenauth Ltd.

package golangcilint

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/cerbos/actions/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/cmd/update-toolbox/tool"
	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/semver"
)

func Update(ctx context.Context, client *github.Client, oldVersion semver.Version) (*tool.Source, error) {
	release, err := client.FindNewerRelease(ctx, github.Repository{Owner: "golangci", Name: "golangci-lint"}, oldVersion)
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
	version := release.Version.Number()

	digestsFile, err := client.DownloadAsset(ctx, release, fmt.Sprintf("golangci-lint-%s-checksums.txt", version))
	if err != nil {
		return nil, err
	}

	digests, err := digests.Parse(digestsFile)
	if err != nil {
		return nil, err
	}

	archives := map[tool.Platform]string{
		tool.LinuxARM64: fmt.Sprintf("golangci-lint-%s-linux-arm64.tar.gz", version),
		tool.LinuxX64:   fmt.Sprintf("golangci-lint-%s-linux-amd64.tar.gz", version),
	}

	source := &tool.Source{
		Version:     release.Version,
		Downloads:   make(map[tool.Platform]tool.Download, len(archives)),
		PostInstall: []string{"golangci-lint", "version"},
	}

	for platform, archive := range archives {
		digest, ok := digests[archive]
		if !ok {
			return nil, fmt.Errorf("missing digest for %s", archive)
		}

		asset, err := release.Asset(archive)
		if err != nil {
			return nil, err
		}

		if digest != asset.Digest {
			return nil, fmt.Errorf("digest mismatch for %s", archive)
		}

		source.Downloads[platform] = tool.Download{
			URL:     asset.URL,
			Digest:  asset.Digest,
			Extract: path.Join(strings.TrimSuffix(archive, ".tar.gz"), "golangci-lint"),
		}
	}

	return source, nil
}
