// Copyright 2026 Zenauth Ltd.

package helmfile

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "helmfile", Name: "helmfile"},
	Verify:      verify,
	PostInstall: []string{"helmfile", "--version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (updater.Installations, error) {
	version := release.Version.Number()

	installations := updater.Installations{
		platform.DarwinARM64: {
			Asset:   fmt.Sprintf("helmfile_%s_darwin_arm64.tar.gz", version),
			Extract: "helmfile",
		},
		platform.LinuxARM64: {
			Asset:   fmt.Sprintf("helmfile_%s_linux_arm64.tar.gz", version),
			Extract: "helmfile",
		},
		platform.LinuxX64: {
			Asset:   fmt.Sprintf("helmfile_%s_linux_amd64.tar.gz", version),
			Extract: "helmfile",
		},
	}

	digestsAsset := fmt.Sprintf("helmfile_%s_checksums.txt", version)

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
