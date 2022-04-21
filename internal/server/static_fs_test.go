package server

import (
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/v3/assert"
)

func TestStaticFileSystemOpensFile(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("ui/dashboard.html")
	assert.NilError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard.html")
	assert.NilError(t, err)

	stat, err := f.Stat()
	assert.NilError(t, err)
	assert.Equal(t, stat.Name(), "dashboard.html")
}

func TestStaticFileSystemAppendDotHtml(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("ui/dashboard.html")
	assert.NilError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard")
	assert.NilError(t, err)

	stat, err := f.Stat()
	assert.NilError(t, err)
	assert.Equal(t, stat.Name(), "dashboard.html")
}

func TestStaticFileSystemExists(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("ui/dashboard/foo")
	assert.NilError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	exists := sfs.Exists("/", "/dashboard")
	assert.Equal(t, exists, true)
}

func TestStaticFileSystemExistsAppendDotHtml(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("ui/dashboard/foo.html")
	assert.NilError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	exists := sfs.Exists("/", "/dashboard/foo")
	assert.Equal(t, exists, true)
}
