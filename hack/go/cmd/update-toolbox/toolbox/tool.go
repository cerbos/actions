// Copyright 2021-2026 Zenauth Ltd.

package toolbox

import (
	"context"
	"fmt"
	"time"

	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/semver"
)

type Tool struct {
	Repo                    github.Repository
	FindNewerReleaseOptions []github.FindNewerReleaseOption
	Verify                  func(context.Context, *Clients, *github.Release) (Installations, error)
	PostInstall             []string
}

type Installations map[Platform]Installation

type Installation struct {
	Asset   string
	Extract string
}

func Update(ctx context.Context, clients *Clients, tool Tool, oldVersion semver.Version) (*Source, error) {
	release, err := clients.GitHub.FindNewerRelease(ctx, tool.Repo, oldVersion, tool.FindNewerReleaseOptions...)
	if release == nil || err != nil {
		return nil, err
	}

	installations, err := tool.Verify(ctx, clients, release)
	if err != nil {
		return nil, fmt.Errorf("failed to verify %s: %w", release, err)
	}

	source := &Source{
		Tag:         release.Tag,
		Version:     release.Version,
		Released:    normalizeTimestamp(release.Created),
		Updated:     normalizeTimestamp(time.Now()),
		Downloads:   make(map[Platform]Download, len(installations)),
		PostInstall: tool.PostInstall,
	}

	for platform, installation := range installations {
		asset, err := release.Asset(installation.Asset)
		if err != nil {
			return nil, err
		}

		source.Downloads[platform] = Download{
			URL:     asset.URL,
			Digest:  asset.Digest,
			Extract: installation.Extract,
		}
	}

	return source, nil
}

func normalizeTimestamp(timestamp time.Time) time.Time {
	return timestamp.UTC().Truncate(time.Second)
}
