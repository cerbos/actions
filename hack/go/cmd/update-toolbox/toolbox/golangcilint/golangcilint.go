// Copyright 2026 Zenauth Ltd.

package golangcilint

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/hack/go/pkg/github"
)

var Tool = toolbox.Tool{
	Repo:        github.Repository{Owner: "golangci", Name: "golangci-lint"},
	Verify:      verify,
	PostInstall: []string{"golangci-lint", "version"},
}

func verify(ctx context.Context, clients *toolbox.Clients, release *github.Release) (toolbox.Installations, error) {
	version := release.Version.Number()

	installations := toolbox.Installations{
		toolbox.LinuxARM64: {
			Asset:   fmt.Sprintf("golangci-lint-%s-linux-arm64.tar.gz", version),
			Extract: fmt.Sprintf("golangci-lint-%s-linux-arm64/golangci-lint", version),
		},
		toolbox.LinuxX64: {
			Asset:   fmt.Sprintf("golangci-lint-%s-linux-amd64.tar.gz", version),
			Extract: fmt.Sprintf("golangci-lint-%s-linux-amd64/golangci-lint", version),
		},
	}

	digestsAsset := fmt.Sprintf("golangci-lint-%s-checksums.txt", version)

	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset); err != nil {
		return nil, err
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
