// Copyright 2026 Zenauth Ltd.

package toolbox

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sourcegraph/conc/pool"
	"go.uber.org/multierr"

	"github.com/cerbos/actions/hack/go/pkg/archive"
	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/semver"
	"github.com/cerbos/actions/hack/go/pkg/tempfile"
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
		Downloads:   make(map[Platform]*Download, len(installations)),
		PostInstall: tool.PostInstall,
	}

	downloads := pool.New().WithContext(ctx).WithFailFast()

	for platform, installation := range installations {
		asset, err := release.Asset(installation.Asset)
		if err != nil {
			return nil, err
		}

		download := &Download{
			URL:     asset.URL,
			Extract: installation.Extract,
			Digests: Digests{Asset: asset.Digest},
		}

		if installation.Extract == "" {
			download.Digests.Binary = asset.Digest
		} else {
			downloads.Go(func(ctx context.Context) error {
				download.Digests.Binary, err = binaryDigestFromArchive(ctx, clients.GitHub, release, asset, download.Extract)
				if err != nil {
					return fmt.Errorf("failed to download and extract %s from %s: %w", asset.Name, release, err)
				}
				return nil
			})
		}

		source.Downloads[platform] = download
	}

	return source, downloads.Wait()
}

func normalizeTimestamp(timestamp time.Time) time.Time {
	return timestamp.UTC().Truncate(time.Second)
}

func binaryDigestFromArchive(ctx context.Context, client *github.Client, release *github.Release, asset *github.Asset, path string) (_ digest.SHA256, err error) {
	contents, err := client.DownloadAsset(ctx, release, asset)
	if err != nil {
		return digest.SHA256{}, err
	}

	archiveFile, err := tempfile.Copy(contents)
	if err != nil {
		return digest.SHA256{}, err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(archiveFile))

	binary, err := archive.Extract(archiveFile, path)
	if err != nil {
		return digest.SHA256{}, err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(binary))

	hash := digest.NewHash()
	if _, err := io.Copy(hash, binary); err != nil {
		return digest.SHA256{}, err
	}

	return hash.Digest(), nil
}
