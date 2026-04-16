// Copyright 2026 Zenauth Ltd.

package digest

import (
	"crypto/sha256"
	"errors"
	"hash"
	"io"
)

var ErrMismatch = errors.New("digest mismatch")

type Hash struct {
	hash.Hash
}

func NewHash() *Hash {
	return &Hash{Hash: sha256.New()}
}

func (h *Hash) Digest() (digest SHA256) {
	h.Sum(digest[:0])
	return digest
}

type reader struct {
	io.Reader
	hash   *Hash
	digest SHA256
}

func NewReader(source io.Reader, digest SHA256) io.ReadCloser {
	hash := NewHash()

	return &reader{
		Reader: io.TeeReader(source, hash),
		digest: digest,
		hash:   hash,
	}
}

func (r *reader) Close() error {
	_, err := io.Copy(io.Discard, r.Reader)
	if err != nil {
		return err
	}

	if r.digest != r.hash.Digest() {
		return ErrMismatch
	}

	return nil
}
