package registry

import (
	"net/http"
	"os"
)

type StaticFileSystem struct {
	base http.FileSystem
}

func (sfs StaticFileSystem) Open(name string) (http.File, error) {
	f, err := sfs.base.Open(name)
	if os.IsNotExist(err) {
		if f, err = sfs.base.Open(name + ".html"); err != nil {
			return sfs.base.Open("404.html")
		}
		return f, nil
	}

	if err != nil {
		return nil, err
	}

	return f, nil
}
