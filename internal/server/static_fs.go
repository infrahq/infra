package server

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

type StaticFileSystem struct {
	base http.FileSystem
}

func (sfs StaticFileSystem) Open(name string) (http.File, error) {
	name = path.Join(uiFilePathPrefix, name)
	f, err := sfs.base.Open(name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	if f, err := sfs.base.Open(name + ".html"); err == nil {
		return f, nil
	}

	if os.IsNotExist(err) {
		return nil, err
	}

	return f, nil
}

const uiFilePathPrefix = "ui/static"

func (sfs StaticFileSystem) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		if _, err := sfs.base.Open(path.Join(uiFilePathPrefix, p)); err == nil {
			return true
		}

		if _, err := sfs.base.Open(path.Join(uiFilePathPrefix, p+".html")); err == nil {
			return true
		}
	}

	return false
}
