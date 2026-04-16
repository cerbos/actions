// Copyright 2026 Zenauth Ltd.

package vals

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "helmfile", Name: "vals"},
	Verify:      verify,
	PostInstall: []string{"vals", "version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (updater.Installations, error) {
	version := release.Version.Number()

	installations := updater.Installations{
		platform.DarwinARM64: {
			Asset:   fmt.Sprintf("vals_%s_darwin_arm64.tar.gz", version),
			Extract: "vals",
		},
		platform.LinuxARM64: {
			Asset:   fmt.Sprintf("vals_%s_linux_arm64.tar.gz", version),
			Extract: "vals",
		},
		platform.LinuxX64: {
			Asset:   fmt.Sprintf("vals_%s_linux_amd64.tar.gz", version),
			Extract: "vals",
		},
	}

	digestsAsset := fmt.Sprintf("vals_%s_checksums.txt", version)

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
