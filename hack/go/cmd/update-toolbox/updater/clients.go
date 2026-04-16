// Copyright 2026 Zenauth Ltd.

package updater

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/http"
	"github.com/cerbos/actions/hack/go/pkg/sigstore"
)

type Clients struct {
	GitHub   *github.Client
	HTTP     *http.Client
	Sigstore *sigstore.Client
}

func NewClients(ctx context.Context) (*Clients, error) {
	sigstoreClient, err := sigstore.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Sigstore client: %w", err)
	}

	return &Clients{
		GitHub:   github.NewClient(ctx),
		HTTP:     http.NewClient(),
		Sigstore: sigstoreClient,
	}, nil
}
