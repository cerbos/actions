// Copyright 2021-2026 Zenauth Ltd.

package digests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/cerbos/actions/cmd/update-toolbox/toolbox"
	"github.com/cerbos/actions/internal/github"
)

func Verify(release *github.Release, installations toolbox.Installations, digestsAssetName string) error {
	digestsAsset, err := release.Asset(digestsAssetName)
	if err != nil {
		return err
	}

	digests, err := parseFile(digestsAsset.Contents)
	if err != nil {
		return err
	}

	for _, installation := range installations {
		asset, err := release.Asset(installation.Asset)
		if err != nil {
			return err
		}

		digest, ok := digests[asset.Name]
		if !ok {
			return fmt.Errorf("missing digest for %s", asset.Name)
		}

		if digest != asset.Digest {
			return fmt.Errorf("digest mismatch for %s", asset.Name)
		}
	}

	return nil
}

func parseFile(contents []byte) (map[string]string, error) {
	digests := make(map[string]string)

	for line := range bytes.Lines(contents) {
		file, digest, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse digest %q", line)
		}
		digests[file] = digest
	}

	return digests, nil
}

func parseLine(line []byte) (string, string, error) {
	digest, file, ok := bytes.Cut(bytes.TrimSpace(line), []byte("  "))
	if !ok {
		return "", "", errors.New("missing separator")
	}

	if hex.DecodedLen(len(digest)) != sha256.Size {
		return "", "", errors.New("incorrect digest length")
	}

	if _, err := hex.Decode(make([]byte, sha256.Size), digest); err != nil {
		return "", "", errors.New("invalid digest encoding")
	}

	return string(file), fmt.Sprintf("sha256:%s", digest), nil
}
