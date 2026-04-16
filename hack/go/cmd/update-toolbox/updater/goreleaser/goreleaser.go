// Copyright 2026 Zenauth Ltd.

package goreleaser

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/sigstore"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

const (
	digestsAsset    = "checksums.txt"
	provenanceAsset = "checksums.txt.sigstore.json"
	workflow        = ".github/workflows/release.yml"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "goreleaser", Name: "goreleaser"},
		Verify:      verify,
		PostInstall: []string{"goreleaser", "--version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    "goreleaser_Darwin_arm64.tar.gz",
			Extract: "goreleaser",
		},
		platform.LinuxARM64: {
			Name:    "goreleaser_Linux_arm64.tar.gz",
			Extract: "goreleaser",
		},
		platform.LinuxX64: {
			Name:    "goreleaser_Linux_x86_64.tar.gz",
			Extract: "goreleaser",
		},
	}
)

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset, provenanceAsset); err != nil {
		return nil, err
	}

	bundle, err := sigstore.BundleFromAsset(release, provenanceAsset)
	if err != nil {
		return nil, err
	}

	ref := "refs/tags/" + release.Tag
	if err := clients.Sigstore.Verify(release, workflow, ref, digestsAsset, bundle); err != nil {
		return nil, err
	}

	if err := digests.VerifyRelease(release, assets, digestsAsset); err != nil {
		return nil, err
	}

	return updater.DownloadsFromRelease(release, assets)
}
