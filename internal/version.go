package internal

import (
	"github.com/Masterminds/semver/v3"
)

var (
	Branch = "main"
	// {x-release-please-start-version}
	Version = "0.14.0"
	// {x-release-please-end}
	Prerelease = ""
	Metadata   = "dev"
	Commit     = ""
	Date       = ""
)

// FullVersion returns the full semver version string, however it also increments the patch version if you're working on a pre-release.
// This is because release-please keeps this at the released version, and not the upcoming next version.
// While the next version may not match the patch release, it causes the right behavior for semver version comparisons.
func FullVersion() string {
	v := semver.MustParse(Version)

	metadata := v.Metadata()
	if v.Metadata() == "" && Metadata != "" {
		metadata = Metadata
	}

	if metadata != "" {
		*v = v.IncPatch()
		*v, _ = v.SetMetadata(metadata)
	}

	if v.Prerelease() == "" && Prerelease != "" {
		*v, _ = v.SetPrerelease(Prerelease)
	}

	return v.String()
}
