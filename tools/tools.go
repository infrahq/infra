// +build tools

package tools

//go:generate go install github.com/envoyproxy/protoc-gen-validate
//go:generate go install github.com/goreleaser/goreleaser
//go:generate go install github.com/kevinburke/go-bindata
//go:generate go install github.com/mitchellh/gon/cmd/gon
//go:generate go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go

import (
	_ "github.com/envoyproxy/protoc-gen-validate"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/kevinburke/go-bindata"
	_ "github.com/mitchellh/gon/cmd/gon"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
