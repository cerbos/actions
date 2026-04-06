// Copyright 2021-2026 Zenauth Ltd.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"
	"go.uber.org/multierr"

	"github.com/cerbos/actions/cmd/update-toolbox/tool"
	"github.com/cerbos/actions/cmd/update-toolbox/tool/buf"
	"github.com/cerbos/actions/cmd/update-toolbox/tool/golangcilint"
	"github.com/cerbos/actions/internal/command"
	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/log"
)

const manifestPath = "../../toolbox.json"

var updaters = map[string]tool.Update{
	"buf":           buf.Update,
	"golangci-lint": golangcilint.Update,
}

func main() {
	command.Run(updateToolbox)
}

func updateToolbox(ctx context.Context) error {
	tools, err := readManifest()
	if err != nil {
		return err
	}

	for name := range tools {
		if _, ok := updaters[name]; !ok {
			log.Debug(ctx, "Removing tool", "tool", name)
			delete(tools, name)
		}
	}

	client := github.NewClient(ctx)
	updates := pool.New().WithContext(ctx)
	start := time.Now()
	var mutex sync.RWMutex
	var updated, notUpdated, failed atomic.Int32

	for name, update := range updaters {
		updates.Go(func(ctx context.Context) error {
			ctx = log.With(ctx, "tool", name)

			mutex.RLock()
			oldVersion := tools[name].Version
			mutex.RUnlock()

			start := time.Now()
			source, err := update(ctx, client, oldVersion)
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
			tools[name] = *source
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

	return writeManifest(tools)
}

func readManifest() (tools map[string]tool.Source, err error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tools file for reading: %w", err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(file))

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&tools); err != nil {
		return nil, fmt.Errorf("failed to read tools file: %w", err)
	}

	return tools, nil
}

func writeManifest(tools map[string]tool.Source) (err error) {
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to open tools file for writing: %w", err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(file))

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(tools); err != nil {
		return fmt.Errorf("failed to write tools file: %w", err)
	}

	return nil
}
