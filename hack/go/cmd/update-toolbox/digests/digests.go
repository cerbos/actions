// Copyright 2021-2026 Zenauth Ltd.

package digests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

func Parse(contents []byte) (map[string]string, error) {
	digests := make(map[string]string)

	for line := range bytes.Lines(contents) {
		file, digest, err := parse(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse digest %q", line)
		}
		digests[file] = digest
	}

	return digests, nil
}

func parse(line []byte) (string, string, error) {
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
