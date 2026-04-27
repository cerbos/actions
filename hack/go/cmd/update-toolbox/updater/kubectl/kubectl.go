// Copyright 2026 Zenauth Ltd.

package kubectl

import (
	"context"
	"fmt"

	"github.com/sourcegraph/conc/pool"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var Tool = updater.Tool{
	Repo:        github.Repository{Owner: "kubernetes", Name: "kubernetes"},
	Verify:      verify,
	PostInstall: []string{"kubectl", "version", "--client"},
}

func verify(ctx context.Context, clients *updater.Clients, release *github.Release) (toolbox.Downloads, error) {
	downloads := toolbox.Downloads{
		platform.DarwinARM64: {URL: fmt.Sprintf("https://dl.k8s.io/release/%s/bin/darwin/arm64/kubectl", release.Version)},
		platform.LinuxARM64:  {URL: fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/arm64/kubectl", release.Version)},
		platform.LinuxX64:    {URL: fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/amd64/kubectl", release.Version)},
	}

	digests := pool.New().WithContext(ctx).WithFailFast()

	for platform, download := range downloads {
		digests.Go(func(ctx context.Context) error {
			if err := fetchDigest(ctx, clients, download); err != nil {
				return fmt.Errorf("failed to fetch digest for %s: %w", platform, err)
			}

			return nil
		})
	}

	return downloads, digests.Wait()
}

func fetchDigest(ctx context.Context, clients *updater.Clients, download *toolbox.Download) error {
	digestHex, err := clients.HTTP.GetBytes(ctx, download.URL+".sha256")
	if err != nil {
		return err
	}

	digest, err := digest.Parse(string(digestHex))
	if err != nil {
		return err
	}

	download.Digests = toolbox.Digests{
		Asset:  digest,
		Binary: digest,
	}

	return nil
}
