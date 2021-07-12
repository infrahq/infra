package version

// Version is set on build
// Release build version is the git tag (set in the .goreleaser ldflags and the Dockerfile BUILDVERSION arg)
var Version = "0.0.0-development"
