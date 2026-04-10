// Copyright 2026 Zenauth Ltd.

package helmfile

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/internal/github"
)

var Tool = toolbox.Tool{
	Repo:        github.Repository{Owner: "helmfile", Name: "helmfile"},
	Verify:      verify,
	PostInstall: []string{"helmfile", "--version"},
}

func verify(ctx context.Context, clients *toolbox.Clients, release *github.Release) (toolbox.Installations, error) {
	version := release.Version.Number()

	installations := toolbox.Installations{
		toolbox.LinuxARM64: {
			Asset:   fmt.Sprintf("helmfile_%s_linux_arm64.tar.gz", version),
			Extract: "helmfile",
		},
		toolbox.LinuxX64: {
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
