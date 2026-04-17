// Copyright 2026 Zenauth Ltd.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"

	"github.com/google/renameio/v2"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/multierr"

	"github.com/cerbos/actions"
	"github.com/cerbos/actions/hack/go/pkg/archive"
	"github.com/cerbos/actions/hack/go/pkg/command"
	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/log"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/tempfile"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

func main() {
	command.Run(installTools)
}

func installTools(ctx context.Context) error {
	installs := pool.New().WithContext(ctx)
	for _, tool := range os.Args[1:] {
		source, ok := actions.Toolbox[tool]
		if !ok {
			return fmt.Errorf("unknown tool %q", tool)
		}

		installs.Go(func(ctx context.Context) error {
			ctx = log.With(ctx, "tool", tool, "version", source.Version)
			err := installTool(ctx, tool, source)
			if err != nil {
				log.Error(ctx, "Failed to install tool", "err", err)
			}
			return err
		})
	}

	if err := installs.Wait(); err != nil {
		return errors.New("failed to install tools")
	}

	return nil
}

func installTool(ctx context.Context, tool string, source toolbox.Source) error {
	download, ok := source.Downloads[platform.Current]
	if !ok {
		return fmt.Errorf("no source for platform %s", platform.Current)
	}

	exists, err := checkForExistingInstallation(ctx, tool, download)
	if err != nil {
		return fmt.Errorf("failed to check for existing installation: %w", err)
	}

	if exists {
		return nil
	}

	if err := downloadTool(ctx, tool, download); err != nil {
		return fmt.Errorf("failed to download tool: %w", err)
	}

	log.Info(ctx, "Installed")
	return nil
}

func checkForExistingInstallation(ctx context.Context, tool string, download *toolbox.Download) (_ bool, err error) {
	file, err := os.Open(tool) //nolint:gosec
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Debug(ctx, "File does not exist")
			return false, nil
		}
		return false, err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(file))

	hash := digest.NewHash()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}

	if hash.Digest() == download.Digests.Binary {
		log.Debug(ctx, "File exists and digest matches")
		return true, nil
	}

	log.Debug(ctx, "File exists but digest does not match")
	return false, nil
}

func downloadTool(ctx context.Context, tool string, download *toolbox.Download) (err error) {
	const permissions = 0o755
	file, err := renameio.NewPendingFile(tool, renameio.WithPermissions(permissions))
	if err != nil {
		return fmt.Errorf("failed to create pending file: %w", err)
	}
	defer multierr.AppendFunc(&err, file.Cleanup)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, download.URL, nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request) //nolint:bodyclose
	if err != nil {
		return fmt.Errorf("GET %s: %w", download.URL, err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(response.Body))

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", download.URL, response.StatusCode)
	}

	source := response.Body

	if download.Extract != "" {
		archiveFile, err := tempfile.Copy(digest.NewReader(response.Body, download.Digests.Asset))
		if err != nil {
			return err
		}
		defer multierr.AppendInvoke(&err, multierr.Close(archiveFile))

		source, err = archive.Extract(download, archiveFile)
		if err != nil {
			return err
		}
		defer multierr.AppendInvoke(&err, multierr.Close(source))
	}

	binary := digest.NewReader(source, download.Digests.Binary)

	if _, err := io.Copy(file, binary); err != nil {
		return multierr.Append(err, binary.Close())
	}

	if err := binary.Close(); err != nil {
		return err
	}

	return file.CloseAtomicallyReplace()
}
