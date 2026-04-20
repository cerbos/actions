// Copyright 2026 Zenauth Ltd.

package telepresence

import (
	"context"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "telepresenceio", Name: "telepresence"},
		Verify:      verify,
		PostInstall: []string{"telepresence", "version"},
	}

	assets = updater.AssetsToDownload{
		platform.DarwinARM64: {Name: "telepresence-darwin-arm64"},
		platform.LinuxARM64:  {Name: "telepresence-linux-arm64"},
		platform.LinuxX64:    {Name: "telepresence-linux-amd64"},
	}
)

func verify(_ context.Context, _ *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	return updater.DownloadsFromRelease(release, assets)
}
