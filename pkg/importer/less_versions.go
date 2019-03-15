package importer

import (
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var versionRegex = regexp.MustCompile(`^v(\d+)(alpha|beta)?(\d+)?$`)

func lessVersions(i, j metav1.APIResource) bool {
	iVersion := i.Version
	jVersion := j.Version

	iParts := versionRegex.FindStringSubmatch(iVersion)
	jParts := versionRegex.FindStringSubmatch(jVersion)

	switch {
	case (iParts == nil) && (jParts == nil):
		// sort non-matching alphabetically
		return iVersion < jVersion
	case (iParts == nil) != (jParts == nil):
		// sort matching before non-matching
		return jParts == nil
	}

	iPrerelease := iParts[2]
	jPrerelease := jParts[2]
	if iPrerelease != jPrerelease {
		switch {
		case iPrerelease == "":
			// if without "alpha" or "beta", it comes first.
			return true
		case iPrerelease == "beta" && jPrerelease == "alpha":
			// "beta" comes before "alpha"
			return true
		default:
			return false
		}
	}

	iMajor := iParts[1]
	jMajor := jParts[1]
	if iMajor != jMajor {
		return larger(iMajor, jMajor)
	}

	iMinor := iParts[3]
	jMinor := jParts[3]
	return larger(iMinor, jMinor)
}

// larger returns whether the number represented by i are larger than the one represented by j.
func larger(i, j string) bool {
	switch {
	case len(i) != len(j):
		return len(i) > len(j)
	default:
		return i > j
	}
}
