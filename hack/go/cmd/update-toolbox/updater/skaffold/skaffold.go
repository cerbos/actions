// Copyright 2026 Zenauth Ltd.

package skaffold

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
	"github.com/cerbos/actions/hack/go/pkg/signing"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

// https://github.com/GoogleContainerTools/skaffold/blob/c186fff81c8031cec0927df89aecd52ce6623eb0/KEYS
const publicKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWZrGCUaJJr1H8a36sG4UUoXvlXvZ
wQfk16sxprI2gOJ2vFFggdq3ixF2h4qNBt0kI7ciDhgpwS8t+/960IsIgw==
-----END PUBLIC KEY-----`

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "GoogleContainerTools", Name: "skaffold"},
	Verify:      verify,
	PostInstall: []string{"skaffold", "version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	ecdsa, err := signing.NewECDSA(publicKey)
	if err != nil {
		return nil, err
	}

	downloads := toolbox.Downloads{
		platform.DarwinARM64: {URL: fmt.Sprintf("https://storage.googleapis.com/skaffold/releases/%s/skaffold-darwin-arm64", release.Version)},
		platform.LinuxARM64:  {URL: fmt.Sprintf("https://storage.googleapis.com/skaffold/releases/%s/skaffold-linux-arm64", release.Version)},
		platform.LinuxX64:    {URL: fmt.Sprintf("https://storage.googleapis.com/skaffold/releases/%s/skaffold-linux-amd64", release.Version)},
	}

	metadatas := make([]*downloadMetadata, 0, len(downloads))
	metadataDownloads := pool.New().WithContext(ctx).WithFailFast()
	for _, download := range downloads {
		digestURL := download.URL + ".sha256"
		signatureURL := digestURL + ".sig"

		metadata := &downloadMetadata{
			Download: download,
			Name:     path.Base(download.URL),
		}

		metadatas = append(metadatas, metadata)

		metadataDownloads.Go(func(ctx context.Context) (err error) {
			metadata.DigestFile, err = clients.HTTP.GetBytes(ctx, digestURL)
			return err
		})

		metadataDownloads.Go(func(ctx context.Context) (err error) {
			metadata.Signature, err = clients.HTTP.GetBytes(ctx, signatureURL)
			return err
		})
	}

	if err := metadataDownloads.Wait(); err != nil {
		return nil, err
	}

	for _, metadata := range metadatas {
		if err := metadata.Verify(ecdsa); err != nil {
			return nil, fmt.Errorf("failed to verify %s: %w", metadata.Name, err)
		}
	}

	return downloads, nil
}

type downloadMetadata struct {
	Download   *toolbox.Download
	Name       string
	DigestFile []byte
	Signature  []byte
}

func (m *downloadMetadata) Verify(ecdsa *signing.ECDSA) error {
	if err := ecdsa.Verify(digest.Bytes(m.DigestFile), m.Signature); err != nil {
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
