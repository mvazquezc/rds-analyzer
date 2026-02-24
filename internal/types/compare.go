// Package types defines the core data structures used throughout the RDS Analyzer.
// These types represent validation reports, diffs, and deviation data from the
// kube-compare JSON reports.
package types

// ValidationReport represents the complete output from the kube-compare JSON reports.
// It contains both a summary of validation results and detailed diffs for each CR.
type ValidationReport struct {
	Summary Summary `json:"Summary"`
	Diffs   []Diff  `json:"Diffs"`
}

// Summary contains high-level statistics about the validation run.
type Summary struct {
	// ValidationIssues maps deviation groups to their specific deviations.
	// Note: The JSON key "ValidationIssuses" has a typo in the source data.
	ValidationIssues ValidationIssues `json:"ValidationIssuses"`

	// NumMissing is the count of expected CRs that were not found.
	NumMissing int `json:"NumMissing"`

	// UnmatchedCRS lists CRs found in the cluster that don't match any template.
	UnmatchedCRS []string `json:"UnmatchedCRS"`

	// NumDiffCRs is the count of CRs that have configuration differences.
	NumDiffCRs int `json:"NumDiffCRs"`

	// TotalCRs is the total number of CRs that were scanned.
	TotalCRs int `json:"TotalCRs"`

	// MetadataHash is a hash identifier for the reference metadata used.
	MetadataHash string `json:"MetadataHash"`

	// PatchedCRs is the count of CRs that required patching.
	PatchedCRs int `json:"patchedCRs"`
}

// ValidationIssues is a hierarchical map of validation problems.
// The first level key is a "deviationgroup" (e.g., "optional-ptp-config", "required-sriov").
// The second level key is a specific "deviation" within that group.
type ValidationIssues map[string]map[string]Deviation

// Deviation represents a single validation issue for a component.
// It describes what CRs are affected and provides context about the problem.
type Deviation struct {
	// Msg describes the validation issue (e.g., "Missing required CR").
	Msg string `json:"Msg"`

	// CRs lists the template paths of affected Custom Resources.
	CRs []string `json:"CRs"`

	// CrMetadata provides additional metadata about each CR (optional).
	CrMetadata map[string]CrMetadata `json:"crMetadata,omitempty"`
}

// CrMetadata holds supplementary information about a Custom Resource.
type CrMetadata struct {
	Description string `json:"description"`
}

// Diff represents a configuration difference found in a Custom Resource.
// It captures the raw diff output and context about which CR and template are involved.
type Diff struct {
	// DiffOutput contains the unified diff showing the configuration differences.
	DiffOutput string `json:"DiffOutput"`

	// CorrelatedTemplate is the path to the reference template this CR was compared against.
	CorrelatedTemplate string `json:"CorrelatedTemplate"`

	// CRName is the full identifier of the Custom Resource (e.g., "v1_ConfigMap_namespace_name").
	CRName string `json:"CRName"`

	// Description provides context about what this CR configures.
	Description string `json:"description"`
}

// DiffLine represents a line in a contextual diff view.
type DiffLine struct {
	Content   string // Line content without diff marker
	IsChanged bool   // True if this is a changed line, false if context
}

// DiffCheck is a parsed representation of a diff, categorizing lines by their type.
// This structured format makes it easier to apply rules and generate reports.
type DiffCheck struct {
	// CRName is the Custom Resource identifier.
	CRName string `json:"CRName"`

	// TemplateFileName is the base name of the reference template.
	TemplateFileName string `json:"TemplateFileName"`

	// FoundValue contains values that were found when both expected and found exist
	// (value difference scenario).
	FoundValue []string `json:"FoundValue"`

	// ExpectedValue contains values that were expected when both expected and found exist
	// (value difference scenario).
	ExpectedValue []string `json:"ExpectedValue"`

	// ExpectedNotFound contains lines that were expected but not found in the actual CR.
	ExpectedNotFound []string `json:"ExpectedNotFound"`

	// FoundNotExpected contains lines that were found but not expected based on the template.
	FoundNotExpected []string `json:"FoundNotExpected"`

	// ExpectedWithContext contains expected lines with surrounding context (3-4 lines).
	ExpectedWithContext []DiffLine `json:"ExpectedWithContext,omitempty"`

	// FoundWithContext contains found lines with surrounding context (3-4 lines).
	FoundWithContext []DiffLine `json:"FoundWithContext,omitempty"`
}
