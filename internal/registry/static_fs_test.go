package registry

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestStaticFileSystemOpensFile(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	fs.Create("dashboard.html")

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard.html")
	assert.Equal(t, err, nil)

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, stat.Name(), "dashboard.html")
}

func TestStaticFileSystemAppendDotHtml(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	fs.Create("dashboard.html")

	sfs := &StaticFileSystem{
		base: fs,
	}

	f, err := sfs.Open("dashboard")
	assert.Equal(t, err, nil)

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, stat.Name(), "dashboard.html")
}

func TestStaticFileSystem404IfNotFound(t *testing.T) {
	fs := afero.NewHttpFs(afero.NewMemMapFs())
	fs.Create("404.html")

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
