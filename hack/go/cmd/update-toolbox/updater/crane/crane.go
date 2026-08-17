// Copyright 2026 Zenauth Ltd.

package crane

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
	provenanceAsset = "multiple.intoto.jsonl"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "google", Name: "go-containerregistry"},
		Verify:      verify,
		PostInstall: []string{"crane", "version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    "go-containerregistry_Darwin_arm64.tar.gz",
			Extract: "crane",
		},
		platform.LinuxARM64: {
			Name:    "go-containerregistry_Linux_arm64.tar.gz",
			Extract: "crane",
		},
		platform.LinuxX64: {
			Name:    "go-containerregistry_Linux_x86_64.tar.gz",
			Extract: "crane",
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

	for _, asset := range assets {
		if err := clients.Sigstore.VerifySLSA(release, asset.Name, bundle); err != nil {
			return nil, err
		}
	}

	if err := digests.VerifyRelease(release, assets, digestsAsset); err != nil {
		return nil, err
	}

	return updater.DownloadsFromRelease(release, assets)
}
