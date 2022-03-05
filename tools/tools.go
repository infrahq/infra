//go:build tools
// +build tools

package tools

//go:generate go install github.com/elazarl/go-bindata-assetfs
//go:generate go install github.com/goreleaser/goreleaser
//go:generate go install github.com/kevinburke/go-bindata/go-bindata
//go:generate go install github.com/swaggo/swag/cmd/swag

import (
	_ "github.com/elazarl/go-bindata-assetfs"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/kevinburke/go-bindata"
	_ "github.com/swaggo/swag/cmd/swag"
)
