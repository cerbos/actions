// Copyright 2026 Zenauth Ltd.

package updater

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/hack/go/pkg/github"
	"github.com/cerbos/actions/hack/go/pkg/sigstore"
)

type Clients struct {
	GitHub   *github.Client
	Sigstore *sigstore.Client
}

func NewClients(ctx context.Context) (*Clients, error) {
	githubClient := github.NewClient(ctx)

	sigstoreClient, err := sigstore.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Sigstore client: %w", err)
	}

	return &Clients{
		GitHub:   githubClient,
		Sigstore: sigstoreClient,
	}, nil
}
