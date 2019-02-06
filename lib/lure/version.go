package lure

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	versionRegex *regexp.Regexp
)

const (
	versionRegexStr = `(\d+)(?:[.-](\d+))?(?:[.-](\d+))?(?:[.-](\d+))?[.-]?(.*)`
)

// Version represents a version with four numeric segments and an optional qualifier (e.g 1.2.3-RELEASE)
// Separators can either be '.' or '-'
type Version struct {
	original  string
	segments  []int64
	qualifier string
}

func init() {
	versionRegex = regexp.MustCompile(versionRegexStr)
}

// ParseVersion interprets a string and returns the corresponding Version.
func ParseVersion(versionStr string) (*Version, error) {
	matches := versionRegex.FindStringSubmatch(versionStr)
	if matches == nil {
		return nil, fmt.Errorf("Malformed version: %s", versionStr)
	}

	segments := make([]int64, 4)
	qualifier := ""

	for i, match := range matches[1:] {

		if match != "" {
			val, err := strconv.ParseInt(match, 10, 64)
			if err != nil {
				if qualifier != "" {
					return nil, fmt.Errorf(
						"Error parsing version: %s", err)
				}
				qualifier = match
			} else {
				segments[i] = int64(val)
			}
		}
	}

	return &Version{
		original:  versionStr,
		segments:  segments,
		qualifier: qualifier,
	}, nil
}

// Equals reports whether two Version are equals
func (v *Version) Equals(o *Version) bool {
	return v.Compare(o) == 0
}

// IsGreaterOrEqualThan reports whether v is greater or equal than other
func (v *Version) IsGreaterOrEqualThan(o *Version) bool {
	return v.Compare(o) >= 0
}

// IsGreaterThan reports whether v is greater than other
func (v *Version) IsGreaterThan(o *Version) bool {
	return v.Compare(o) > 0
}

// IsLessThan reports whether v is less than other
func (v *Version) IsLessThan(o *Version) bool {
	return v.Compare(o) < 0
}

// IsLessOrEqualThan reports whether v is less or equal than other
func (v *Version) IsLessOrEqualThan(o *Version) bool {
	return v.Compare(o) <= 0
}

// Compare compares version v and other and returns a negative number
// if v < other and a positive number if v > other
func (v *Version) Compare(other *Version) int {

	for i := range v.segments {
		if v.segments[i] < other.segments[i] {
			return -1
		}
		if v.segments[i] > other.segments[i] {
			return 1
		}
	}
	return strings.Compare(v.qualifier, other.qualifier)
}

func (v *Version) String() string {
	return v.original
}
