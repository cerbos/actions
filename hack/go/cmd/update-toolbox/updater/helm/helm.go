// Copyright 2026 Zenauth Ltd.

package helm

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/sourcegraph/conc/pool"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/semver"
	"github.com/cerbos/actions/hack/go/pkg/signing"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var Tool = updater.Tool{
	Repo: github.Repository{Owner: "helm", Name: "helm"},
	FindNewerReleaseOptions: []github.FindNewerReleaseOption{
		github.VersionConstraint(func(version semver.Version) bool {
			return semver.Compare(version, "v4") < 0
		}),
	},
	Verify:      verify,
	PostInstall: []string{"helm", "version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	pgp, err := signing.NewPGP(publicKeys)
	if err != nil {
		return nil, err
	}

	downloads := toolbox.Downloads{
		platform.DarwinARM64: {
			URL:     fmt.Sprintf("https://get.helm.sh/helm-%s-darwin-arm64.tar.gz", release.Version),
			Extract: "darwin-arm64/helm",
		},
		platform.LinuxARM64: {
			URL:     fmt.Sprintf("https://get.helm.sh/helm-%s-linux-arm64.tar.gz", release.Version),
			Extract: "linux-arm64/helm",
		},
		platform.LinuxX64: {
			URL:     fmt.Sprintf("https://get.helm.sh/helm-%s-linux-amd64.tar.gz", release.Version),
			Extract: "linux-amd64/helm",
		},
	}

	metadatas := make([]*downloadMetadata, 0, len(downloads))
	signatureAssets := make([]string, 0, len(downloads))
	for _, download := range downloads {
		digestURL := download.URL + ".sha256sum"
		signatureAsset := path.Base(digestURL) + ".asc"

		metadatas = append(metadatas, &downloadMetadata{
			Download:       download,
			Name:           path.Base(download.URL),
			DigestURL:      digestURL,
			SignatureAsset: signatureAsset,
		})

		signatureAssets = append(signatureAssets, signatureAsset)
	}

	metadataDownloads := pool.New().WithContext(ctx).WithFailFast()

	metadataDownloads.Go(func(ctx context.Context) error {
		return clients.GitHub.DownloadAssets(ctx, release, signatureAssets...)
	})

	for _, metadata := range metadatas {
		metadataDownloads.Go(func(ctx context.Context) (err error) {
			metadata.DigestFile, err = clients.HTTP.GetBytes(ctx, metadata.DigestURL)
			return err
		})
	}

	if err := metadataDownloads.Wait(); err != nil {
		return nil, err
	}

	for _, metadata := range metadatas {
		if err := metadata.Verify(pgp, release); err != nil {
			return nil, fmt.Errorf("failed to verify %s: %w", metadata.Name, err)
		}
	}

	return downloads, nil
}

type downloadMetadata struct {
	Download       *toolbox.Download
	Name           string
	DigestURL      string
	SignatureAsset string
	DigestFile     []byte
}

func (m *downloadMetadata) Verify(pgp *signing.PGP, release *github.Release) error {
	if err := pgp.Verify(m.DigestFile, release.Assets[m.SignatureAsset].Contents); err != nil {
		return err
	}

	digests, err := digest.ParseFile(m.DigestFile)
	if err != nil {
		return err
	}

	var ok bool
	m.Download.Digests.Asset, ok = digests[m.Name]
	if !ok {
		return errors.New("missing digest")
	}

	return nil
}
