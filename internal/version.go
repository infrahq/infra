package internal

import "github.com/Masterminds/semver/v3"

var (
	Branch = "main"
	// {x-release-please-start-version}
	Version = "0.13.0"
	// {x-release-please-end}
	PrereleaseTag = ""
	Metadata      = "dev"
	Commit        = ""
	Date          = ""
)

// FullVersion returns the full semver version string, however it also increments the patch version if you're working on a pre-release.
// This is because release-please keeps this at the released version, and not the upcoming next version.
// While the next version may not match the patch release, it causes the right behavior for semver version comparisons.
func FullVersion() string {
	v, _ := semver.NewVersion(Version)
	if Metadata == "dev" {
		*v = v.IncPatch()
	}
	*v, _ = v.SetPrerelease(PrereleaseTag)
	*v, _ = v.SetMetadata(Metadata)

	return v.String()
}
