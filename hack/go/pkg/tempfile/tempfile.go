// Copyright 2026 Zenauth Ltd.

package tempfile

import (
	"io"
	"os"

	"go.uber.org/multierr"
)

type File struct {
	*os.File
	Size int64
}

func Create() (*File, error) {
	file, err := os.CreateTemp("", "cerbos-actions-*")
	return &File{File: file}, err
}

func (f File) Close() error {
	return multierr.Append(f.File.Close(), os.Remove(f.Name()))
}

func Copy(source io.ReadCloser) (file *File, err error) {
	defer multierr.AppendInvoke(&err, multierr.Close(source))

	file, err = Create()
	if err != nil {
		return file, err
	}
	defer func() {
		if err != nil {
			multierr.AppendInvoke(&err, multierr.Close(file))
		}
	}()

	file.Size, err = io.Copy(file, source)
	if err != nil {
		return file, err
	}

	if err := file.Sync(); err != nil {
		return file, err
	}

	_, err = file.Seek(0, io.SeekStart)
	return file, err
}
