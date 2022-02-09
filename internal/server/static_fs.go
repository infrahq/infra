package server

import (
	"net/http"
	"os"
	"strings"
)

type StaticFileSystem struct {
	base http.FileSystem
}

func (sfs StaticFileSystem) Open(name string) (http.File, error) {
	f, err := sfs.base.Open(name)
	if err != nil && !os.IsNotExist(err) {
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

func (sfs StaticFileSystem) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		_, err := sfs.base.Open(p)
		return err == nil
	}

	return false
}
