// Copyright 2026 Zenauth Ltd.

package buf

import (
	"context"
	"errors"

	"aead.dev/minisign"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
)

const (
	digestsAsset   = "sha256.txt"
	signatureAsset = "sha256.txt.minisig"
)

var (
	Tool = updater.Tool{
		Repo:        github.Repository{Owner: "bufbuild", Name: "buf"},
		Verify:      verify,
		PostInstall: []string{"buf", "--version"},
	}

	installations = updater.Installations{
		platform.DarwinARM64: {Asset: "buf-Darwin-arm64"},
		platform.LinuxARM64:  {Asset: "buf-Linux-aarch64"},
		platform.LinuxX64:    {Asset: "buf-Linux-x86_64"},
	}

	publicKey minisign.PublicKey
)

func init() {
	// https://buf.build/docs/cli/installation/#verifying-a-release
	if err := publicKey.UnmarshalText([]byte("RWQ/i9xseZwBVE7pEniCNjlNOeeyp4BQgdZDLQcAohxEAH5Uj5DEKjv6")); err != nil {
		panic(err)
	}
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (updater.Installations, error) {
	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset, signatureAsset); err != nil {
		return nil, err
	}

	if !minisign.Verify(publicKey, release.Assets[digestsAsset].Contents, release.Assets[signatureAsset].Contents) {
		return nil, errors.New("invalid signature")
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
