package registry

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticFileSystemOpensFile(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("dashboard.html")
	require.NoError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard.html")
	assert.Equal(t, err, nil)

	stat, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, stat.Name(), "dashboard.html")
}

func TestStaticFileSystemAppendDotHtml(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("dashboard.html")
	require.NoError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard")
	assert.Equal(t, err, nil)

	stat, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, stat.Name(), "dashboard.html")
}

func TestStaticFileSystem404IfNotFound(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	_, err := fs.Create("404.html")
	require.NoError(t, err)

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard")
	assert.Equal(t, err, nil)

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, stat.Name(), "404.html")
}
