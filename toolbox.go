// Copyright 2026 Zenauth Ltd.

package actions

import (
	"bytes"

	"github.com/cerbos/actions/hack/go/pkg/toolbox"

	_ "embed"
)

var (
	Toolbox toolbox.Manifest

	//go:embed toolbox.json
	manifest []byte
)

func init() {
	var err error
	Toolbox, err = toolbox.Parse(bytes.NewReader(manifest))
	if err != nil {
		panic(err)
	}
}
