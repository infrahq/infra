// +build tools

package tools

//go:generate go install github.com/kevinburke/go-bindata
//go:generate go install github.com/mitchellh/gon/cmd/gon

import (
	_ "github.com/kevinburke/go-bindata"
	_ "github.com/mitchellh/gon/cmd/gon"
)
