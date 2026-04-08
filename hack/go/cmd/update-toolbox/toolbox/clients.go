// Copyright 2021-2026 Zenauth Ltd.

package toolbox

import (
	"context"
	"fmt"

	"github.com/cerbos/actions/internal/github"
	"github.com/cerbos/actions/internal/sigstore"
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
