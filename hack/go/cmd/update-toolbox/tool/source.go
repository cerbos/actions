// Copyright 2021-2026 Zenauth Ltd.

package tool

import (
	"context"

	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/semver"
)

type Update func(ctx context.Context, client *github.Client, oldVersion semver.Version) (*Source, error)

type Source struct {
	Version     semver.Version        `json:"version"`
	Downloads   map[Platform]Download `json:"downloads"`
	PostInstall []string              `json:"postInstall"`
}

type Platform string

const (
	LinuxARM64 = "linux/arm64"
	LinuxX64   = "linux/x64"
)

type Download struct {
	URL     string `json:"url"`
	Digest  string `json:"digest"`
	Extract string `json:"extract,omitempty"`
}
