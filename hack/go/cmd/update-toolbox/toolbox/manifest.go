// Copyright 2026 Zenauth Ltd.

package toolbox

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.uber.org/multierr"

	"github.com/cerbos/actions/internal/semver"
)

const manifestPath = "../../toolbox.json"

type Source struct {
	Tag         string                `json:"-"`
	Version     semver.Version        `json:"version"`
	Released    time.Time             `json:"released"`
	Updated     time.Time             `json:"updated"`
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

func Read() (manifest map[string]Source, err error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tools file for reading: %w", err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(file))

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to read tools file: %w", err)
	}

	return manifest, nil
}

func Write(manifest map[string]Source) (err error) {
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to open tools file for writing: %w", err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(file))

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to write tools file: %w", err)
	}

	return nil
}
