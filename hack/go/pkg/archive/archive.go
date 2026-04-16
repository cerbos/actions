// Copyright 2026 Zenauth Ltd.

package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"

	"go.uber.org/multierr"
)

type extractor struct {
	io.Reader
	io.Closer
}

func Extract(source io.Reader, path string) (_ io.ReadCloser, err error) {
	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			multierr.AppendInvoke(&err, multierr.Close(gzipReader))
		}
	}()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%q not found in archive", path)
		} else if err != nil {
			return nil, err
		}

		if header.Name != path {
			continue
		}

		if header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("%q is not a regular file in archive", path)
		}

		return extractor{Reader: tarReader, Closer: gzipReader}, nil
	}
}
