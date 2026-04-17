// Copyright 2026 Zenauth Ltd.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/buf"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/crane"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/flipt"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/golangcilint"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/goreleaser"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/grype"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/helm"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/helmfile"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/just"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/oras"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/reimage"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/rmz"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/skaffold"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/syft"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/telepresence"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/terraform"
	"github.com/cerbos/actions/hack/go/cmd/update-toolbox/updater/vals"
	"github.com/cerbos/actions/hack/go/pkg/command"
	"github.com/cerbos/actions/hack/go/pkg/log"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

var tools = map[string]updater.Tool{
	"buf":           buf.Tool,
	"crane":         crane.Tool,
	"flipt":         flipt.Tool,
	"golangci-lint": golangcilint.Tool,
	"goreleaser":    goreleaser.Tool,
	"grype":         grype.Tool,
	"helm":          helm.Tool,
	"helmfile":      helmfile.Tool,
	"just":          just.Tool,
	"oras":          oras.Tool,
	"reimage":       reimage.Tool,
	"rmz":           rmz.Tool,
	"skaffold":      skaffold.Tool,
	"syft":          syft.Tool,
	"telepresence":  telepresence.Tool,
	"terraform":     terraform.Tool,
	"vals":          vals.Tool,
}

func main() {
	command.Run(updateToolbox)
}

func updateToolbox(ctx context.Context) error {
	clients, err := updater.NewClients(ctx)
	if err != nil {
		return err
	}

	manifest, err := toolbox.Read()
	if err != nil {
		return err
	}

	for name := range manifest {
		if _, ok := tools[name]; !ok {
			log.Debug(ctx, "Removing tool", "tool", name)
			delete(manifest, name)
		}
	}

	updates := pool.NewWithResults[string]().WithContext(ctx)
	start := time.Now()
	var mutex sync.RWMutex
	var updated, notUpdated, failed atomic.Int32

	for name, tool := range tools {
		updates.Go(func(ctx context.Context) (string, error) {
			ctx = log.With(ctx, "tool", name)

			mutex.RLock()
			oldVersion := manifest[name].Version
			mutex.RUnlock()

			start := time.Now()
			source, err := updater.Update(ctx, clients, tool, oldVersion)
			ctx = log.With(ctx, "duration", time.Since(start))
			if err != nil {
				failed.Add(1)
				log.Error(ctx, "Update failed", "err", err)
				return "", err
			}

			if source == nil {
				notUpdated.Add(1)
				log.Info(ctx, "No update available")
				return "", nil
			}

			mutex.Lock()
			manifest[name] = *source
			mutex.Unlock()

			updated.Add(1)
			log.Info(ctx, "Updated", "version", source.Version)
			return fmt.Sprintf("%s | %s | [%s](https://github.com/%s/releases/tag/%s)", name, oldVersion, source.Version, tool.Repo, source.Tag), nil
		})
	}

	rows, err := updates.Wait()
	log.Info(ctx, "Completed", "duration", time.Since(start), "updated", updated.Load(), "notUpdated", notUpdated.Load(), "failed", failed.Load())
	if err != nil {
		switch n := failed.Load(); n {
		case 0:
			return err
		case 1:
			return errors.New("1 update failed")
		default:
			return fmt.Errorf("%d updates failed", n)
		}
	}

	if os.Getenv("CI") == "true" && updated.Load() > 0 {
		fmt.Fprintln(os.Stdout, "Update | from | to\n---|---|---")
		for _, row := range rows {
			if row != "" {
				fmt.Fprintln(os.Stdout, row)
			}
		}
	}

	return toolbox.Write(manifest)
}
