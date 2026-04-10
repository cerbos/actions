// Copyright 2026 Zenauth Ltd.

package sigstore

import (
	"bytes"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/cerbos/actions/hack/go/pkg/github"
)

const issuer = "https://token.actions.githubusercontent.com"

type Client struct {
	verifier *verify.Verifier
}

func NewClient() (*Client, error) {
	trustedRoot, err := root.FetchTrustedRoot()
	if err != nil {
		return nil, err
	}

	verifier, err := verify.NewVerifier(
		trustedRoot,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithObserverTimestamps(1),
		verify.WithTransparencyLog(1),
	)
	if err != nil {
		return nil, err
	}

	return &Client{verifier: verifier}, nil
}

func (c *Client) Verify(release *github.Release, workflow, ref, artifactAssetName, bundleAssetName string) error {
	artifactAsset, err := release.Asset(artifactAssetName)
	if err != nil {
		return err
	}

	artifact := bytes.NewReader(artifactAsset.Contents)

	bundleAsset, err := release.Asset(bundleAssetName)
	if err != nil {
		return err
	}

	var bundle bundle.Bundle
	if err := bundle.UnmarshalJSON(bundleAsset.Contents); err != nil {
		return err
	}

	san := fmt.Sprintf("https://github.com/%s/%s@%s", release.Repo, workflow, ref)
	identity, err := verify.NewShortCertificateIdentity(issuer, "", san, "")
	if err != nil {
		return err
	}

	_, err = c.verifier.Verify(&bundle, verify.NewPolicy(verify.WithArtifact(artifact), verify.WithCertificateIdentity(identity)))
	return err
}
