package infra

import (
	"net/http"
	"os"
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
		return sfs.base.Open("404.html")
	}

	return f, nil
}
