// Package rules implements the rule-based evaluation engine for RDS analysis.
package rules

import (
	"fmt"
	"strconv"
	"strings"
)

// OCPVersion represents an OpenShift Container Platform version (e.g., 4.19).
// Versions are compared numerically by major and minor components.
type OCPVersion struct {
	Major int
	Minor int
}

// ParseOCPVersion parses a version string like "4.19" into an OCPVersion.
// Returns an error if the format is invalid or contains non-numeric components.
func ParseOCPVersion(s string) (OCPVersion, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return OCPVersion{}, fmt.Errorf("empty version string")
	}

	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return OCPVersion{}, fmt.Errorf("invalid version format %q: expected Major.Minor (e.g., 4.19)", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return OCPVersion{}, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return OCPVersion{}, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}

	return OCPVersion{Major: major, Minor: minor}, nil
}

// Compare compares two OCPVersion instances.
// Returns:
//   - -1 if v < other
//   - 0 if v == other
//   - 1 if v > other
func (v OCPVersion) Compare(other OCPVersion) int {
	if v.Major < other.Major {
		return -1
	}
	if v.Major > other.Major {
		return 1
	}
	if v.Minor < other.Minor {
		return -1
	}
	if v.Minor > other.Minor {
		return 1
	}
	return 0
}

// String returns the string representation of the version (e.g., "4.19").
func (v OCPVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// IsZero returns true if the version is unset (zero value).
func (v OCPVersion) IsZero() bool {
	return v.Major == 0 && v.Minor == 0
}
