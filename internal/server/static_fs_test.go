package server

import (
	"net/http"
	"testing"
	"testing/fstest"

	"gotest.tools/v3/assert"
)

func TestStaticFileSystemOpensFile(t *testing.T) {
	fs := fstest.MapFS{
		"ui/static/foo.html": {
			Data: []byte("<html></html>"),
		},
	}

	sfs := &StaticFileSystem{
		base: http.FS(fs),
	}

	f, err := sfs.Open("foo.html")
	assert.NilError(t, err)

	stat, err := f.Stat()
	assert.NilError(t, err)
	assert.Equal(t, stat.Name(), "foo.html")
}

func TestStaticFileSystemAppendDotHtml(t *testing.T) {
	fs := fstest.MapFS{
		"ui/static/foo.html": {
			Data: []byte("<html></html>"),
		},
	}

	sfs := &StaticFileSystem{
		base: http.FS(fs),
	}

	f, err := sfs.Open("foo")
	assert.NilError(t, err)

	stat, err := f.Stat()
	assert.NilError(t, err)
	assert.Equal(t, stat.Name(), "foo.html")
}

func TestStaticFileSystemExists(t *testing.T) {
	fs := fstest.MapFS{
		"ui/static/foo.html": {
			Data: []byte("<html></html>"),
		},
	}

	sfs := &StaticFileSystem{
		base: http.FS(fs),
	}

	exists := sfs.Exists("/", "/foo")
	assert.Equal(t, exists, true)
}

func TestStaticFileSystemExistsAppendDotHtml(t *testing.T) {
	fs := fstest.MapFS{
		"ui/static/foo.html": {
			Data: []byte("<html></html>"),
		},
	}

	sfs := &StaticFileSystem{
		base: http.FS(fs),
	}

	exists := sfs.Exists("/", "/foo.html")
	assert.Equal(t, exists, true)
}
