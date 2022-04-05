package server

import (
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestStaticFileSystemOpensFile(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("dashboard.html")
	assert.NilError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard.html")
	assert.Check(t, is.DeepEqual(err, nil))

	stat, err := f.Stat()
	assert.NilError(t, err)
	assert.Check(t, is.Equal(stat.Name(), "dashboard.html"))
}

func TestStaticFileSystemAppendDotHtml(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("dashboard.html")
	assert.NilError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard")
	assert.Check(t, is.DeepEqual(err, nil))

	stat, err := f.Stat()
	assert.NilError(t, err)
	assert.Check(t, is.Equal(stat.Name(), "dashboard.html"))
}
