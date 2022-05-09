package server

import (
	"strconv"
	"strings"
)

type Version struct {
	Major      uint8
	Minor      uint8
	Patch      uint8
	Prerelease string
}

func NewVersion(ver string) Version {
	parts := strings.Split(ver, ".")
	v := Version{}
	if len(parts) > 0 {
		v.Major = parseuint8(parts[0])
	}
	if len(parts) > 1 {
		v.Minor = parseuint8(parts[1])
	}
	if len(parts) > 2 {
		patchParts := strings.Split(parts[2], "-")
		v.Patch = parseuint8(patchParts[0])
		if len(patchParts) > 1 {
			v.Prerelease = patchParts[1]
		}
	}
	return v
}

func parseuint8(s string) uint8 {
	i, _ := strconv.ParseUint(s, 10, 8)
	return uint8(i)
}

func (v Version) GreaterThan(ver Version) bool {
	if v.Major < ver.Major {
		return false
	}
	if v.Major > ver.Major {
		return true
	}
	if v.Minor < ver.Minor {
		return false
	}
	if v.Minor > ver.Minor {
		return true
	}
	if v.Patch < ver.Patch {
		return false
	}
	if v.Patch > ver.Patch {
		return true
	}
	if len(v.Prerelease) > 0 {
		return true
	}
	return false
}

func (v Version) GreaterThanStr(ver string) bool {
	return v.GreaterThan(NewVersion(ver))
}
