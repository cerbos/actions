// Copyright 2026 Zenauth Ltd.

package toolbox

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/multierr"

	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/platform"
	"github.com/cerbos/actions/hack/go/pkg/semver"
)

const manifestPath = "toolbox.json"

type Manifest map[string]Source

type Source struct {
	Tag         string         `json:"-"`
	Version     semver.Version `json:"version"`
	Released    time.Time      `json:"released"`
	Updated     time.Time      `json:"updated"`
	Downloads   Downloads      `json:"downloads"`
	PostInstall []string       `json:"postInstall"`
}

type Downloads map[platform.Platform]*Download

type Download struct {
	URL     string  `json:"url"`
	Extract string  `json:"extract,omitempty"`
	Digests Digests `json:"digests"`
}

type Digests struct {
	Asset  digest.SHA256 `json:"asset"`
	Binary digest.SHA256 `json:"binary"`
}

func Read() (_ Manifest, err error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tools file for reading: %w", err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(file))

	return Parse(file)
}

func Parse(file io.Reader) (manifest Manifest, _ error) {
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to read tools file: %w", err)
	}

	return manifest, nil
}

func Write(manifest Manifest) (err error) {
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
