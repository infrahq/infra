// Package version is used check what the verson variable was set to when the running build was created.
package internal

// Version is set on build, it should follow sematic versioning. Release build version is the git tag (set in the .goreleaser ldflags and the Dockerfile BUILDVERSION arg).
var Version = "0.0.0-development"
