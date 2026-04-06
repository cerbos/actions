// Copyright 2021-2026 Zenauth Ltd.

package semver

import (
	"encoding/json"
	"log/slog"
	"strings"

	"golang.org/x/mod/semver"
)

type Version string

func (v Version) IsValid() bool {
	return semver.IsValid(string(v))
}

func (v Version) LogValue() slog.Value {
	return slog.StringValue(v.Number())
}

func (v Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Number())
}

func (v Version) Number() string {
	return strings.TrimPrefix(string(v), "v")
}

func (v Version) String() string {
	return string(v)
}

func (v *Version) UnmarshalJSON(data []byte) error {
	var number string
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}

	*v = Version("v" + number)
	return nil
}

func Compare(a, b Version) int {
	return semver.Compare(string(a), string(b))
}
