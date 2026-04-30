// Copyright 2026 Zenauth Ltd.

package protoc

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "protocolbuffers", Name: "protobuf"},
	Verify:      verify,
	PostInstall: []string{"protoc", "--version"},
}

func verify(_ context.Context, _ *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	version := release.Version.Number()

	assets := updater.AssetsToDownload{
		platform.DarwinARM64: {
			Name:    fmt.Sprintf("protoc-%s-osx-aarch_64.zip", version),
			Extract: "bin/protoc",
		},
		platform.LinuxARM64: {
			Name:    fmt.Sprintf("protoc-%s-linux-aarch_64.zip", version),
			Extract: "bin/protoc",
		},
		platform.LinuxX64: {
			Name:    fmt.Sprintf("protoc-%s-linux-x86_64.zip", version),
			Extract: "bin/protoc",
		},
	}

	return updater.DownloadsFromRelease(release, assets)
}
