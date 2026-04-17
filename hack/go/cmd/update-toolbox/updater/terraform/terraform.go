// Copyright 2026 Zenauth Ltd.

package terraform

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/sourcegraph/conc/pool"
	"go.uber.org/multierr"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/signing"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "hashicorp", Name: "terraform"},
	Verify:      verify,
	PostInstall: []string{"terraform", "version"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	urlPrefix := fmt.Sprintf("https://releases.hashicorp.com/terraform/%[1]s/terraform_%[1]s_", release.Version.Number())

	digests, err := fetchDigests(ctx, clients, urlPrefix)
	if err != nil {
		return nil, err
	}

	downloads := toolbox.Downloads{
		platform.DarwinARM64: {
			URL:     urlPrefix + "darwin_arm64.zip",
			Extract: "terraform",
		},
		platform.LinuxARM64: {
			URL:     urlPrefix + "linux_arm64.zip",
			Extract: "terraform",
		},
		platform.LinuxX64: {
			URL:     urlPrefix + "linux_amd64.zip",
			Extract: "terraform",
		},
	}

	for _, download := range downloads {
		name := path.Base(download.URL)
		var ok bool
		download.Digests.Asset, ok = digests[name]
		if !ok {
			return nil, fmt.Errorf("missing digest for %s", name)
		}
	}

	return downloads, nil
}

func fetchDigests(ctx context.Context, clients *updater.Clients, urlPrefix string) (digest.Digests, error) {
	pgp, err := signing.NewPGP(publicKeys)
	if err != nil {
		return nil, err
	}

	type Download struct {
		URL      string
		Contents []byte
	}

	digests := &Download{URL: urlPrefix + "SHA256SUMS"}
	signature := &Download{URL: digests.URL + ".sig"}

	downloads := pool.New().WithContext(ctx).WithFailFast()

	for _, download := range []*Download{digests, signature} {
		downloads.Go(func(ctx context.Context) (err error) {
			responseBody, err := clients.HTTP.Get(ctx, download.URL)
			if err != nil {
				return err
			}
			defer multierr.AppendInvoke(&err, multierr.Close(responseBody))

			download.Contents, err = io.ReadAll(responseBody)
			if err != nil {
				return fmt.Errorf("failed to download %s: %w", download.URL, err)
			}

			return nil
		})
	}

	if err := downloads.Wait(); err != nil {
		return nil, err
	}

	if err := pgp.Verify(digests.Contents, signature.Contents); err != nil {
		return nil, err
	}

	return digest.ParseFile(digests.Contents)
}
