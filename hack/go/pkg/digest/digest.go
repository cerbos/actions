// Copyright 2026 Zenauth Ltd.

package digest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

const prefix = "sha256:"

type SHA256 [sha256.Size]byte

func (d SHA256) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

func (d SHA256) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d SHA256) String() string {
	return prefix + hex.EncodeToString(d[:])
}

func (d *SHA256) UnmarshalJSON(data []byte) (err error) {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	*d, err = Parse(encoded)
	return err
}

func Parse(encoded string) (digest SHA256, _ error) {
	hexEncoded := strings.TrimPrefix(encoded, prefix)

	if hex.DecodedLen(len(hexEncoded)) != sha256.Size {
		return digest, fmt.Errorf("invalid digest %q", encoded)
	}

	_, err := hex.Decode(digest[:], []byte(hexEncoded))
	if err != nil {
		return digest, fmt.Errorf("invalid digest %q: %w", encoded, err)
	}

	return digest, nil
}
