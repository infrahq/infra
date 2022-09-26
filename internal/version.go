package internal

import (
	"github.com/Masterminds/semver/v3"
)

var (
	Branch  = "main"
	Version = "99.99.99999"
	Commit  = ""
	Date    = ""
)

// FullVersion returns the full semantic version string. FullVersion panics if
// the version string is not a valid semantic version.
func FullVersion() string {
	return semver.MustParse(Version).String()
}
