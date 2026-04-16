// Copyright 2026 Zenauth Ltd.

package goreleaser

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
)

const (
	digestsAsset   = "checksums.txt"
	signatureAsset = "checksums.txt.sigstore.json"
	workflow       = ".github/workflows/release.yml"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "goreleaser", Name: "goreleaser"},
		Verify:      verify,
		PostInstall: []string{"goreleaser", "--version"},
	}

	installations = updater.Installations{
		platform.DarwinARM64: {
			Asset:   "goreleaser_Darwin_arm64.tar.gz",
			Extract: "goreleaser",
		},
		platform.LinuxARM64: {
			Asset:   "goreleaser_Linux_arm64.tar.gz",
			Extract: "goreleaser",
		},
		platform.LinuxX64: {
			Asset:   "goreleaser_Linux_x86_64.tar.gz",
			Extract: "goreleaser",
		},
	}
)

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (updater.Installations, error) {
	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset, signatureAsset); err != nil {
		return nil, err
	}

	ref := "refs/tags/" + release.Tag
	if err := clients.Sigstore.Verify(release, workflow, ref, digestsAsset, signatureAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
