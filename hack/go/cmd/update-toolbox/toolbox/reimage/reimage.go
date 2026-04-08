// Copyright 2021-2026 Zenauth Ltd.

package reimage

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/internal/github"
)

const digestsAsset = "checksums.txt"

var Tool = toolbox.Tool{
	Repo:        github.Repository{Owner: "cerbos", Name: "reimage"},
	Verify:      verify,
	PostInstall: []string{"reimage", "-V"},
}

func verify(ctx context.Context, clients *toolbox.Clients, release *github.Release) (toolbox.Installations, error) {
	version := release.Version.Number()

	installations := toolbox.Installations{
		toolbox.LinuxARM64: {
			Asset:   fmt.Sprintf("reimage_%s_Linux_arm64.tar.gz", version),
			Extract: "reimage",
		},
		toolbox.LinuxX64: {
			Asset:   fmt.Sprintf("reimage_%s_Linux_x86_64.tar.gz", version),
			Extract: "reimage",
		},
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
