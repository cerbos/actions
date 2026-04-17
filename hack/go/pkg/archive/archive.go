// Copyright 2026 Zenauth Ltd.

package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"go.uber.org/multierr"

	"github.com/cerbos/actions/hack/go/pkg/tempfile"
	"github.com/cerbos/actions/hack/go/pkg/toolbox"
)

func Extract(download *toolbox.Download, archive *tempfile.File) (io.ReadCloser, error) {
	switch {
	case strings.HasSuffix(download.URL, ".tar.gz"):
		return extractTarGz(archive, download.Extract)
	case strings.HasSuffix(download.URL, ".zip"):
		return extractZip(archive, download.Extract)
	default:
		return nil, fmt.Errorf("unknown archive format %s", download.URL)
	}
}

type readCloser struct {
	io.Reader
	io.Closer
}

func extractTarGz(source *tempfile.File, path string) (_ io.ReadCloser, err error) {
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

		return readCloser{Reader: tarReader, Closer: gzipReader}, nil
	}
}

func extractZip(source *tempfile.File, path string) (io.ReadCloser, error) {
	zipReader, err := zip.NewReader(source, source.Size)
	if err != nil {
		return nil, err
	}

	return zipReader.Open(path)
}
