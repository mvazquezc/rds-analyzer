// Package analyzer provides the core analysis orchestration for RDS validation.
// It coordinates the rule engine, parsing, and report generation.
package analyzer

import (
	"fmt"
	"io"

	"github.com/openshift-kni/rds-analyzer/internal/report"
	"github.com/openshift-kni/rds-analyzer/internal/rules"
	"github.com/openshift-kni/rds-analyzer/internal/types"
)

// Analyzer orchestrates the RDS validation analysis.
// It loads rules, evaluates validation reports, and generates output.
type Analyzer struct {
	ruleEngine *rules.Engine
}

// New creates a new Analyzer with rules loaded from the specified file.
// If version is non-empty, rules are evaluated against that OCP version.
func New(rulesFile, version string) (*Analyzer, error) {
	engine, err := createRuleEngine(rulesFile, version)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rule engine: %w", err)
	}

	return &Analyzer{ruleEngine: engine}, nil
}

// createRuleEngine creates an engine with or without version specification.
func createRuleEngine(rulesFile, version string) (*rules.Engine, error) {
	if version != "" {
		return rules.NewEngineWithVersion(rulesFile, version)
	}
	return rules.NewEngine(rulesFile)
}

// Analyze processes a validation report and writes results to the given writer.
// The format parameter determines output type: "text" or "html".
// The mode parameter determines output mode: "simple" or "reporting".
func (a *Analyzer) Analyze(w io.Writer, validationReport types.ValidationReport, format, mode string) error {
	if mode == "reporting" {
		return a.generateReportingOutput(w, validationReport)
	}

	return a.generateFormattedOutput(w, validationReport, format)
}

// generateReportingOutput generates output in reporting mode.
func (a *Analyzer) generateReportingOutput(w io.Writer, validationReport types.ValidationReport) error {
	generator := report.NewReportingGenerator(a.ruleEngine)
	return generator.Generate(w, validationReport)
}

// generateFormattedOutput generates output in the specified format.
func (a *Analyzer) generateFormattedOutput(w io.Writer, validationReport types.ValidationReport, format string) error {
	switch format {
	case "text":
		generator := report.NewTextGenerator(a.ruleEngine)
		return generator.Generate(w, validationReport)
	case "html":
		generator := report.NewHTMLGenerator(a.ruleEngine)
		return generator.Generate(w, validationReport)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// GetTargetVersion returns the OCP version being used for analysis.
func (a *Analyzer) GetTargetVersion() string {
	if v := a.ruleEngine.GetTargetVersion(); !v.IsZero() {
		return v.String()
	}
	return ""
}
