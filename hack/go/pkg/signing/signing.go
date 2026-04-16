// Copyright 2026 Zenauth Ltd.

package signing

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"aead.dev/minisign"

	"github.com/cerbos/actions/hack/go/pkg/digest"
)

var ErrVerificationFailed = errors.New("failed to verify signature")

type ECDSA ecdsa.PublicKey

func NewECDSA(publicKey string) (*ECDSA, error) {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return nil, errors.New("failed to parse PEM-encoded public key")
	}

	pkix, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DER-encoded public key: %w", err)
	}

	key, ok := pkix.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected ECDSA public key, got %T", pkix)
	}

	return (*ECDSA)(key), nil
}

func (e *ECDSA) Verify(digest digest.SHA256, signature []byte) error {
	if !ecdsa.VerifyASN1((*ecdsa.PublicKey)(e), digest[:], signature) {
		return ErrVerificationFailed
	}

	return nil
}

type Minisign minisign.PublicKey

func NewMinisign(publicKey string) (Minisign, error) {
	var minisignPublicKey minisign.PublicKey
	err := minisignPublicKey.UnmarshalText([]byte(publicKey))
	return Minisign(minisignPublicKey), err
}

func (m Minisign) Verify(message, signature []byte) error {
	if !minisign.Verify(minisign.PublicKey(m), message, signature) {
		return ErrVerificationFailed
	}

	return nil
}
