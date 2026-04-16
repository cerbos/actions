// Copyright 2026 Zenauth Ltd.

package updater

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
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/semver"
	"github.com/cerbos/actions/hack/go/pkg/tempfile"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

type Tool struct {
	Repo                    github.Repository
	FindNewerReleaseOptions []github.FindNewerReleaseOption
	Verify                  func(context.Context, *Clients, *github.Release) (toolbox.Downloads, error)
	PostInstall             []string
}

func Update(ctx context.Context, clients *Clients, tool Tool, oldVersion semver.Version) (*toolbox.Source, error) {
	release, err := clients.GitHub.FindNewerRelease(ctx, tool.Repo, oldVersion, tool.FindNewerReleaseOptions...)
	if release == nil || err != nil {
		return nil, err
	}

	downloads, err := tool.Verify(ctx, clients, release)
	if err != nil {
		return nil, fmt.Errorf("failed to verify %s: %w", release, err)
	}

	source := &toolbox.Source{
		Tag:         release.Tag,
		Version:     release.Version,
		Released:    normalizeTimestamp(release.Created),
		Updated:     normalizeTimestamp(time.Now()),
		Downloads:   downloads,
		PostInstall: tool.PostInstall,
	}

	return source, setBinaryDigests(ctx, clients, source)
}

func normalizeTimestamp(timestamp time.Time) time.Time {
	return timestamp.UTC().Truncate(time.Second)
}

func setBinaryDigests(ctx context.Context, clients *Clients, source *toolbox.Source) error {
	downloads := pool.New().WithContext(ctx).WithFailFast()

	for _, download := range source.Downloads {
		if download.Extract == "" {
			download.Digests.Binary = download.Digests.Asset
		} else {
			downloads.Go(func(ctx context.Context) (err error) {
				return setBinaryDigest(ctx, clients, download)
			})
		}
	}

	return downloads.Wait()
}

func setBinaryDigest(ctx context.Context, clients *Clients, download *toolbox.Download) (err error) {
	responseBody, err := clients.HTTP.Get(ctx, download.URL)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(responseBody))

	archiveFile, err := tempfile.Copy(digest.NewReader(responseBody, download.Digests.Asset))
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(archiveFile))

	binary, err := archive.Extract(archiveFile, download.Extract)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(binary))

	hash := digest.NewHash()
	if _, err := io.Copy(hash, binary); err != nil {
		return err
	}

	download.Digests.Binary = hash.Digest()
	return nil
}

type AssetsToDownload map[platform.Platform]AssetToDownload

type AssetToDownload struct {
	Name    string
	Extract string
}

func DownloadsFromRelease(release *github.Release, assets AssetsToDownload) (toolbox.Downloads, error) {
	downloads := make(toolbox.Downloads, len(assets))

	for platform, download := range assets {
		asset, err := release.Asset(download.Name)
		if err != nil {
			return nil, err
		}

		downloads[platform] = &toolbox.Download{
			URL:     asset.URL,
			Extract: download.Extract,
			Digests: toolbox.Digests{Asset: asset.Digest},
		}
	}

	return downloads, nil
}
