// Copyright 2026 Zenauth Ltd.

package digest

import (
	"bytes"
	"errors"
	"fmt"
)

type Digests map[string]SHA256

func ParseFile(contents []byte) (Digests, error) {
	digests := make(map[string]SHA256)

	for line := range bytes.Lines(contents) {
		file, digest, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse digest %q: %w", line, err)
		}
		digests[file] = digest
	}

	return digests, nil
}

func parseLine(line []byte) (string, SHA256, error) {
	encoded, file, ok := bytes.Cut(bytes.TrimSpace(line), []byte("  "))
	if !ok {
		return "", SHA256{}, errors.New("missing separator")
	}

	digest, err := Parse(string(encoded))
	return string(file), digest, err
}
