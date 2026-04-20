package rules

import (
	"fmt"
	"strings"
)

// RegexValidationError is returned when one or more regex or value_regex patterns
// in rules YAML fail Go's regexp.Compile. Analysis must not run until these are fixed.
type RegexValidationError struct {
	Warnings []string
}

// Error implements error.
func (e *RegexValidationError) Error() string {
	var b strings.Builder
	b.WriteString("invalid regex pattern(s) in rules file(s):\n\n")
	for i, w := range e.Warnings {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, w)
	}
	b.WriteString("\nFix these patterns before running analysis.")
	return b.String()
}
