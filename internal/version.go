package internal

var (
	Branch = "main"
	// {x-release-please-start-version}
	Version = "0.13.0"
	// {x-release-please-end}
	PrereleaseTag = "dev"
	Commit        = ""
	Date          = ""
)

func FullVersion() string {
	if len(PrereleaseTag) > 0 {
		return Version + "-" + PrereleaseTag
	}
	return Version
}
