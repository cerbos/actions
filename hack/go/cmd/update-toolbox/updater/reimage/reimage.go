// Copyright 2026 Zenauth Ltd.

package reimage

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
)

const digestsAsset = "checksums.txt"

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "cerbos", Name: "reimage"},
	Verify:      verify,
	PostInstall: []string{"reimage", "-V"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (updater.Installations, error) {
	version := release.Version.Number()

	installations := updater.Installations{
		platform.DarwinARM64: {
			Asset:   fmt.Sprintf("reimage_%s_Darwin_arm64.tar.gz", version),
			Extract: "reimage",
		},
		platform.LinuxARM64: {
			Asset:   fmt.Sprintf("reimage_%s_Linux_arm64.tar.gz", version),
			Extract: "reimage",
		},
		platform.LinuxX64: {
			Asset:   fmt.Sprintf("reimage_%s_Linux_x86_64.tar.gz", version),
			Extract: "reimage",
		},
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
