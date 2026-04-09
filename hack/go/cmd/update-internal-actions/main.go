// Copyright 2021-2026 Zenauth Ltd.

package main

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/sourcegraph/conc/pool"

	"github.com/cerbos/actions/internal/command"
	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/log"
)

var (
	referencePattern = regexp.MustCompile(`uses: cerbos/actions/([^@]+)@([0-9a-f]{40})`)
	repo             = github.Repository{Owner: "cerbos", Name: "actions"}
)

func main() {
	command.Run(updateInternalActions)
}

func updateInternalActions(ctx context.Context) error {
	loads := pool.NewWithResults[*Action]().WithContext(ctx).WithFailFast()

	for path, err := range findActions(ctx) {
		if err != nil {
			return err
		}

		loads.Go(func(ctx context.Context) (*Action, error) {
			return loadAction(ctx, path)
		})
	}

	actions, err := loads.Wait()
	if err != nil {
		return err
	}

	client := github.NewClient(ctx)
	versions := make(map[string]*ActionVersion)
	updates := pool.New().WithContext(ctx).WithFailFast()
	i := 0
	for _, action := range actions {
		if action == nil {
			continue
		}

		actions[i] = action
		i++

		for name := range action.References {
			if _, ok := versions[name]; !ok {
				version := &ActionVersion{Name: name}
				versions[name] = version
				updates.Go(func(ctx context.Context) error {
					return version.Update(ctx, client)
				})
			}
		}
	}
	actions = actions[:i]

	if err := updates.Wait(); err != nil {
		return err
	}

	saves := pool.New().WithContext(ctx)

	for _, action := range actions {
		saves.Go(func(ctx context.Context) error {
			return action.Save(ctx, versions)
		})
	}

	return saves.Wait()
}

func findActions(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		output, err := exec.CommandContext(ctx, "git", "ls-files", "-z", ":/*/action.yaml").Output()
		if err != nil {
			yield("", fmt.Errorf("failed to list action files: %w", err))
			return
		}

		for path := range bytes.SplitSeq(output, []byte{0}) {
			if len(path) > 0 && !yield(string(path), nil) {
				return
			}
		}
	}
}

type Action struct {
	References map[string][]int
	Name       string
	Path       string
	Contents   []byte
}

func loadAction(ctx context.Context, path string) (*Action, error) {
	name := filepath.Base(filepath.Dir(path))
	ctx = log.With(ctx, "action", name)

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	matches := referencePattern.FindAllSubmatchIndex(contents, -1)
	if len(matches) == 0 {
		log.Debug(ctx, "No references found")
		return nil, nil
	}

	const (
		nameStart    = 2
		nameEnd      = 3
		versionStart = 4
	)

	action := &Action{
		Name:       name,
		Path:       path,
		Contents:   contents,
		References: make(map[string][]int, len(matches)),
	}
	for _, match := range matches {
		reference := string(contents[match[nameStart]:match[nameEnd]])
		action.References[reference] = append(action.References[reference], match[versionStart])
		log.Debug(ctx, "Found reference", "reference", reference)
	}

	return action, nil
}

func (a *Action) Save(ctx context.Context, versions map[string]*ActionVersion) error {
	ctx = log.With(ctx, "action", a.Name)

	for reference, indexes := range a.References {
		version := versions[reference].Version
		log.Debug(ctx, "Updated reference", "reference", reference, "version", version)
		for _, index := range indexes {
			copy(a.Contents[index:], version)
		}
	}

	const permissions = 0o644
	return os.WriteFile(a.Path, a.Contents, permissions)
}

type ActionVersion struct {
	Name    string
	Version string
}

func (v *ActionVersion) Update(ctx context.Context, client *github.Client) error {
	commit, err := client.FindLatestCommitForPath(ctx, repo, v.Name)
	if err != nil {
		return err
	}

	log.Debug(ctx, "Found latest commit", "action", v.Name, "commit", commit)
	v.Version = commit.SHA
	return nil
}
