// Copyright 2026 Zenauth Ltd.

package shellcheck

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "koalaman", Name: "shellcheck"},
	Verify:      verify,
	PostInstall: []string{"shellcheck", "--version"},
}

func verify(_ context.Context, _ *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	extract := fmt.Sprintf("shellcheck-%s/shellcheck", release.Version)

	assets := updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    fmt.Sprintf("shellcheck-%s.darwin.aarch64.tar.gz", release.Version),
			Extract: extract,
		},
		platform.LinuxARM64: {
			Name:    fmt.Sprintf("shellcheck-%s.linux.aarch64.tar.gz", release.Version),
			Extract: extract,
		},
		platform.LinuxX64: {
			Name:    fmt.Sprintf("shellcheck-%s.linux.x86_64.tar.gz", release.Version),
			Extract: extract,
		},
	}

	return updater.DownloadsFromRelease(release, assets)
}
