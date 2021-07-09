package version

import "strings"

const versionPrefix = "v"

// Version is set on build
// Release build version is the git tag (set in the .goreleaser ldflags and the Dockerfile BUILDVERSION arg)
var Version = "development"

// GetFormattedVersion returns the current version in a standard format with the 'v' prefix removed
func GetFormattedVersion() string {
	return strings.TrimPrefix(Version, versionPrefix)
}
