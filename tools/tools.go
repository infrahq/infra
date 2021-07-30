// +build tools

package tools

//go:generate go install github.com/elazarl/go-bindata-assetfs
//go:generate go install github.com/envoyproxy/protoc-gen-validate
//go:generate go install github.com/goreleaser/goreleaser
//go:generate go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
//go:generate go install github.com/grpc-ecosystem/protoc-gen-grpc-gateway-ts
//go:generate go install github.com/kevinburke/go-bindata
//go:generate go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go

import (
	_ "github.com/elazarl/go-bindata-assetfs"
	_ "github.com/envoyproxy/protoc-gen-validate"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/protoc-gen-grpc-gateway-ts"
	_ "github.com/kevinburke/go-bindata"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
