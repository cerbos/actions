// Copyright 2021-2026 Zenauth Ltd.

package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox/buf"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox/golangcilint"
	"github.com/cerbos/actions/cmd/update-toolbox/toolbox/just"
	"github.com/cerbos/actions/internal/command"
	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/log"
)

var tools = map[string]toolbox.Tool{
	"buf":           buf.Tool,
	"golangci-lint": golangcilint.Tool,
	"just":          just.Tool,
}

func main() {
	command.Run(updateToolbox)
}

func updateToolbox(ctx context.Context) error {
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

	client := github.NewClient(ctx)
	updates := pool.New().WithContext(ctx)
	start := time.Now()
	var mutex sync.RWMutex
	var updated, notUpdated, failed atomic.Int32

	for name, tool := range tools {
		updates.Go(func(ctx context.Context) error {
			ctx = log.With(ctx, "tool", name)

			mutex.RLock()
			oldVersion := manifest[name].Version
			mutex.RUnlock()

			start := time.Now()
			source, err := toolbox.Update(ctx, client, tool, oldVersion)
			ctx = log.With(ctx, "duration", time.Since(start))
			if err != nil {
				failed.Add(1)
				log.Error(ctx, "Update failed", "err", err)
				return err
			}

			if source == nil {
				notUpdated.Add(1)
				log.Info(ctx, "No update available")
				return nil
			}

			mutex.Lock()
			manifest[name] = *source
			mutex.Unlock()

			updated.Add(1)
			log.Info(ctx, "Updated", "version", source.Version)
			return nil
		})
	}

	err = updates.Wait()
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

	return toolbox.Write(manifest)
}
