// Copyright 2026 Zenauth Ltd.

package sigstore

import (
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

func (c *Client) Verify(release *github.Release, workflow, ref, artifactAssetName string, bundle *bundle.Bundle) error {
	san := fmt.Sprintf("%s/%s@%s", release.Repo.URL(), workflow, ref)
	identity, err := verify.NewShortCertificateIdentity(issuer, "", san, "")
	if err != nil {
		return err
	}

	_, err = c.verify(release, artifactAssetName, bundle, identity)
	return err
}

func (c *Client) VerifySLSA(release *github.Release, artifactAssetName string, bundle *bundle.Bundle) error {
	identity, err := verify.NewShortCertificateIdentity(issuer, "", "", `^https://github\.com/slsa-framework/slsa-github-generator/\.github/workflows/generator_generic_slsa3\.yml@refs/tags/v\d+\.\d+\.\d+$`)
	if err != nil {
		return err
	}

	result, err := c.verify(release, artifactAssetName, bundle, identity)
	if err != nil {
		return err
	}

	cert := result.Signature.Certificate

	if repo := cert.SourceRepositoryURI; repo != release.Repo.URL() {
		return fmt.Errorf("unexpected source repository %q", repo)
	}

	if ref := cert.SourceRepositoryRef; ref != "refs/tags/"+release.Tag {
		return fmt.Errorf("unexpected source ref %q", ref)
	}

	return nil
}

func (c *Client) verify(release *github.Release, assetName string, bundle *bundle.Bundle, identity verify.CertificateIdentity) (*verify.VerificationResult, error) {
	asset, err := release.Asset(assetName)
	if err != nil {
		return nil, err
	}

	return c.verifier.Verify(bundle, verify.NewPolicy(
		verify.WithArtifactDigest("sha256", asset.Digest[:]),
		verify.WithCertificateIdentity(identity)),
	)
}

func BundleFromAsset(release *github.Release, assetName string) (*bundle.Bundle, error) {
	asset, err := release.Asset(assetName)
	if err != nil {
		return nil, err
	}

	var bundle bundle.Bundle
	if err := bundle.UnmarshalJSON(asset.Contents); err != nil {
		return nil, err
	}

	return &bundle, nil
}
