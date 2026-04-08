// Copyright 2021-2026 Zenauth Ltd.

package just

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/semver"
)

const digestsAsset = "SHA256SUMS"

var Tool = toolbox.Tool{
	Repo: github.Repository{Owner: "casey", Name: "just"},
	FindNewerReleaseOptions: []github.FindNewerReleaseOption{
		github.VersionFromTag(func(tag string) semver.Version {
			return semver.Version("v" + tag)
		}),
	},
	Verify:      verify,
	PostInstall: []string{"just", "--version"},
}

func verify(ctx context.Context, clients *toolbox.Clients, release *github.Release) (toolbox.Installations, error) {
	version := release.Version.Number()

	installations := toolbox.Installations{
		toolbox.LinuxARM64: {
			Asset:   fmt.Sprintf("just-%s-aarch64-unknown-linux-musl.tar.gz", version),
			Extract: "just",
		},
		toolbox.LinuxX64: {
			Asset:   fmt.Sprintf("just-%s-x86_64-unknown-linux-musl.tar.gz", version),
			Extract: "just",
		},
	}

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
