// Copyright 2026 Zenauth Ltd.

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/multierr"

	"github.com/cerbos/actions/hack/go/pkg/command"
	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/log"
)

type Action struct {
	Source        Source
	Target        string
	ExcludeInputs []string
}

type Source struct {
	Repo github.Repository
	Path string
}

type Input struct { //nolint:govet
	Description        string `yaml:"description"`
	DeprecationMessage string `yaml:"deprecationMessage,omitempty"`
	Required           bool   `yaml:"required"`
	Default            string `yaml:"default,omitempty"`
}

var actions = []Action{
	{
		Target: "buf",
		Source: Source{
			Repo: github.Repository{Owner: "bufbuild", Name: "buf-action"},
			Path: "action.yml",
		},
		ExcludeInputs: []string{
			"checksum",
			"version",
		},
	},
	{
		Target: "create-pull-request",
		Source: Source{
			Repo: github.Repository{Owner: "peter-evans", Name: "create-pull-request"},
			Path: "action.yml",
		},
		ExcludeInputs: []string{
			"author",
			"body",
			"body-path",
			"branch",
			"branch-token",
			"branch-suffix",
			"commit-message",
			"committer",
			"delete-branch",
			"maintainer-can-modify",
			"push-to-fork",
			"sign-commits",
			"signoff",
			"title",
			"token",
		},
	},
	{
		Target: "golangci-lint",
		Source: Source{
			Repo: github.Repository{Owner: "golangci", Name: "golangci-lint-action"},
			Path: "action.yml",
		},
		ExcludeInputs: []string{
			"install-mode",
			"install-only",
			"version",
			"version-file",
		},
	},
}

func main() {
	command.Run(wrapActions)
}

func wrapActions(ctx context.Context) error {
	client := github.NewClient(ctx)
	tasks := pool.New().WithContext(ctx).WithFailFast()

	for _, action := range actions {
		tasks.Go(func(ctx context.Context) error {
			return wrapAction(log.With(ctx, "action", action.Target), client, action)
		})
	}

	return tasks.Wait()
}

func wrapAction(ctx context.Context, client *github.Client, action Action) error {
	path := filepath.Join(action.Target, "action.yaml")

	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to load action %s: %w", action.Target, err)
	}

	commit, err := findPinnedCommit(ctx, action.Source, contents)
	if err != nil {
		return fmt.Errorf("failed to parse action %s: %w", action.Target, err)
	}

	inputs, err := fetchInputs(ctx, client, action.Source, commit)
	if err != nil {
		return err
	}

	if err := writeInputs(ctx, path, contents, inputs, action.ExcludeInputs); err != nil {
		return fmt.Errorf("failed to save action %s: %w", action.Target, err)
	}

	log.Info(ctx, "Done")
	return nil
}

func findPinnedCommit(ctx context.Context, source Source, contents []byte) (string, error) {
	uses := path.Dir(path.Join(source.Repo.String(), source.Path)) + "@"

	var action struct {
		Runs struct {
			Steps []struct {
				Uses string `yaml:"uses"`
			} `yaml:"steps"`
		} `yaml:"runs"`
	}

	if err := yaml.UnmarshalContext(ctx, contents, &action); err != nil {
		return "", err
	}

	for _, step := range action.Runs.Steps {
		if version, ok := strings.CutPrefix(step.Uses, uses); ok {
			return version, nil
		}
	}

	return "", errors.New("no matching `uses` step found")
}

func fetchInputs(ctx context.Context, client *github.Client, source Source, commit string) (map[string]Input, error) {
	file, err := client.DownloadFile(ctx, source.Repo, commit, source.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s from %s@%s: %w", source.Path, source.Repo, commit, err)
	}
	defer file.Close()

	var action struct {
		Inputs map[string]Input `yaml:"inputs"`
	}

	if err := yaml.NewDecoder(file).DecodeContext(ctx, &action); err != nil {
		return nil, fmt.Errorf("failed to parse %s from %s@%s: %w", source.Path, source.Repo, commit, err)
	}

	return action.Inputs, nil
}

func writeInputs(ctx context.Context, path string, contents []byte, inputs map[string]Input, exclude []string) (err error) {
	for _, input := range exclude {
		if _, ok := inputs[input]; !ok {
			return fmt.Errorf("unknown input %s", input)
		}

		delete(inputs, input)
	}

	inputsBytes, err := yaml.MarshalContext(ctx, inputs, yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(file))

	type State struct {
		Insert    iter.Seq[[]byte]
		Next      *State
		Delimiter []byte
		Copy      bool
	}

	end := &State{Copy: true}

	wrapUsesWithEnd := &State{
		Delimiter: []byte("        # wrap:uses.with:end\n"),
		Insert: func(yield func([]byte) bool) {
			for _, input := range slices.Sorted(maps.Keys(inputs)) {
				if !yield(fmt.Appendf(nil, "        %[1]s: ${{ inputs.%[1]s }}\n", input)) {
					return
				}
			}
		},
		Next: end,
	}

	wrapUsesWithStart := &State{
		Delimiter: []byte("        # wrap:uses.with:start\n"),
		Copy:      true,
		Next:      wrapUsesWithEnd,
	}

	wrapInputsEnd := &State{
		Delimiter: []byte("  # wrap:inputs:end\n"),
		Insert: func(yield func([]byte) bool) {
			for line := range bytes.Lines(inputsBytes) {
				if !yield([]byte("  ")) || !yield(line) {
					return
				}
			}
		},
		Next: wrapUsesWithStart,
	}

	state := &State{
		Delimiter: []byte("  # wrap:inputs:start\n"),
		Copy:      true,
		Next:      wrapInputsEnd,
	}

	for line := range bytes.Lines(contents) {
		foundDelimiter := state.Delimiter != nil && bytes.Equal(line, state.Delimiter)

		if foundDelimiter || state.Copy {
			if _, err := file.Write(line); err != nil {
				return err
			}
		}

		if foundDelimiter {
			state = state.Next

			if state.Insert != nil {
				for insertion := range state.Insert {
					if _, err := file.Write(insertion); err != nil {
						return err
					}
				}
			}
		}
	}

	if state != end {
		return fmt.Errorf("%q not found", bytes.TrimSpace(state.Delimiter))
	}

	return nil
}
