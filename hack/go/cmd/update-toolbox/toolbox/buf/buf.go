// Copyright 2026 Zenauth Ltd.

package buf

import (
	"context"
	"errors"

	"aead.dev/minisign"

	"github.com/cerbos/actions/cmd/update-toolbox/digests"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/internal/github"
)

const (
	digestsAsset   = "sha256.txt"
	signatureAsset = "sha256.txt.minisig"
)

var (
	Tool = toolbox.Tool{
		Repo:        github.Repository{Owner: "bufbuild", Name: "buf"},
		Verify:      verify,
		PostInstall: []string{"buf", "--version"},
	}

	installations = toolbox.Installations{
		toolbox.LinuxARM64: {Asset: "buf-Linux-aarch64"},
		toolbox.LinuxX64:   {Asset: "buf-Linux-x86_64"},
	}

	publicKey minisign.PublicKey
)

func init() {
	// https://buf.build/docs/cli/installation/#verifying-a-release
	if err := publicKey.UnmarshalText([]byte("RWQ/i9xseZwBVE7pEniCNjlNOeeyp4BQgdZDLQcAohxEAH5Uj5DEKjv6")); err != nil {
		panic(err)
	}
}

func verify(ctx context.Context, clients *toolbox.Clients, release *github.Release) (toolbox.Installations, error) {
	if err := clients.GitHub.DownloadAssets(ctx, release, digestsAsset, signatureAsset); err != nil {
		return nil, err
	}

	if !minisign.Verify(publicKey, release.Assets[digestsAsset].Contents, release.Assets[signatureAsset].Contents) {
		return nil, errors.New("invalid signature")
	}

	return installations, digests.Verify(release, installations, digestsAsset)
}
