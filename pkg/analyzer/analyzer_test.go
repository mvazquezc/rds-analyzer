package analyzer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openshift-kni/rds-analyzer/pkg/types"
)

// testRulesYAML contains a minimal rules configuration for testing.
const testRulesYAML = `
version: "1.0"
description: "Test Rules"

settings:
  default_impact: "NeedsReview"
  default_severity: "MEDIUM"

label_annotation_rules:
  labels: []
  annotations: []
  default_impact: "NotADeviation"
  default_comment: "Labels and annotations are acceptable"

rules:
  - id: "R001-test"
    description: "Test rule"
    match:
      crName: "*"
    conditions:
      - type: "ExpectedFound"
        contains: "name:"
        impact: "NotImpacting"
        comment: "Name changes are acceptable"
`

func createTestRulesFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "rules.yaml")
	if err := os.WriteFile(rulesFile, []byte(testRulesYAML), 0644); err != nil {
		t.Fatalf("Failed to create test rules file: %v", err)
	}
	return rulesFile
}

func TestNew_ValidRulesFile(t *testing.T) {
	rulesFile := createTestRulesFile(t)

	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected analyzer, got nil")
	}
}

func TestNew_WithVersion(t *testing.T) {
	rulesFile := createTestRulesFile(t)

	a, err := New(rulesFile, "4.19")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected analyzer, got nil")
	}
	if a.GetTargetVersion() != "4.19" {
		t.Errorf("expected version 4.19, got %s", a.GetTargetVersion())
	}
}

func TestNew_InvalidRulesFile(t *testing.T) {
	_, err := New("/nonexistent/rules.yaml", "")
	if err == nil {
		t.Fatal("expected error for nonexistent rules file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to initialize rule engine") {
		t.Errorf("expected error message about rule engine, got: %v", err)
	}
}

func TestNew_InvalidVersion(t *testing.T) {
	rulesFile := createTestRulesFile(t)

	// Invalid version format should fail
	_, err := New(rulesFile, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid version, got nil")
	}
}

func TestAnalyze_TextFormat(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 0,
			NumDiffCRs: 1,
			TotalCRs:   10,
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  name: test\n+  name: changed",
				CorrelatedTemplate: "test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
				Description:        "Test CR",
			},
		},
	}

	var buf bytes.Buffer
	err = a.Analyze(&buf, report, "text", "simple")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestAnalyze_HTMLFormat(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 0,
			NumDiffCRs: 1,
			TotalCRs:   5,
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  value: old\n+  value: new",
				CorrelatedTemplate: "test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
			},
		},
	}

	var buf bytes.Buffer
	err = a.Analyze(&buf, report, "html", "simple")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML output")
	}
}

func TestAnalyze_ReportingMode(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 0,
			NumDiffCRs: 1,
			TotalCRs:   5,
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  config: expected\n+  config: found",
				CorrelatedTemplate: "test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
			},
		},
	}

	var buf bytes.Buffer
	err = a.Analyze(&buf, report, "text", "reporting")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty output for reporting mode")
	}
}

func TestAnalyze_UnsupportedFormat(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{}

	var buf bytes.Buffer
	err = a.Analyze(&buf, report, "unsupported", "simple")
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("expected error about unsupported format, got: %v", err)
	}
}

func TestGetTargetVersion_NoVersion(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	// Without specifying a version, GetTargetVersion might return empty or default
	version := a.GetTargetVersion()
	// Just ensure it doesn't panic and returns a string
	_ = version
}

func TestGetTargetVersion_WithVersion(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "4.20")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	version := a.GetTargetVersion()
	if version != "4.20" {
		t.Errorf("expected version 4.20, got %s", version)
	}
}

func TestAnalyze_EmptyReport(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{}

	var buf bytes.Buffer
	err = a.Analyze(&buf, report, "text", "simple")
	if err != nil {
		t.Fatalf("Analyze should handle empty report: %v", err)
	}
}

func TestNewFromBytes_ValidRules(t *testing.T) {
	a, err := NewFromBytes([]byte(testRulesYAML), "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected analyzer, got nil")
	}
}

func TestNewFromBytes_WithVersion(t *testing.T) {
	a, err := NewFromBytes([]byte(testRulesYAML), "4.19")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.GetTargetVersion() != "4.19" {
		t.Errorf("expected version 4.19, got %s", a.GetTargetVersion())
	}
}

func TestNewFromBytes_InvalidYAML(t *testing.T) {
	_, err := NewFromBytes([]byte("not: [valid: yaml"), "")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestNewFromBytes_InvalidVersion(t *testing.T) {
	_, err := NewFromBytes([]byte(testRulesYAML), "invalid")
	if err == nil {
		t.Fatal("expected error for invalid version, got nil")
	}
}

func TestNewFromBytes_InvalidRegex(t *testing.T) {
	invalidRules := `
version: "1.0"
settings:
  default_impact: "NeedsReview"
rules:
  - id: "bad-rule"
    match: {}
    conditions:
      - type: "Any"
        regex: "[unclosed"
        impact: "Impacting"
        comment: "bad regex"
`
	_, err := NewFromBytes([]byte(invalidRules), "")
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
	if !strings.Contains(err.Error(), "failed to initialize rule engine") {
		t.Errorf("expected rule engine error, got: %v", err)
	}
}

func TestNewFromBytes_ProducesSameOutputAsNew(t *testing.T) {
	rulesFile := createTestRulesFile(t)

	fileAnalyzer, err := New(rulesFile, "4.20")
	if err != nil {
		t.Fatalf("Failed to create file analyzer: %v", err)
	}

	bytesAnalyzer, err := NewFromBytes([]byte(testRulesYAML), "4.20")
	if err != nil {
		t.Fatalf("Failed to create bytes analyzer: %v", err)
	}

	report := types.ValidationReport{
		Summary: types.Summary{
			NumDiffCRs: 1,
			TotalCRs:   5,
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  name: test\n+  name: changed",
				CorrelatedTemplate: "test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
			},
		},
	}

	var fileBuf, bytesBuf bytes.Buffer
	if err := fileAnalyzer.Analyze(&fileBuf, report, "text", "simple"); err != nil {
		t.Fatalf("File analyzer failed: %v", err)
	}
	if err := bytesAnalyzer.Analyze(&bytesBuf, report, "text", "simple"); err != nil {
		t.Fatalf("Bytes analyzer failed: %v", err)
	}

	if fileBuf.String() != bytesBuf.String() {
		t.Error("file-based and bytes-based analyzers produced different output")
	}
}

func TestNewFromBytes_HTMLFormat(t *testing.T) {
	a, err := NewFromBytes([]byte(testRulesYAML), "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{
		Summary: types.Summary{NumDiffCRs: 1, TotalCRs: 5},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  value: old\n+  value: new",
				CorrelatedTemplate: "test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
			},
		},
	}

	var buf bytes.Buffer
	if err := a.Analyze(&buf, report, "html", "simple"); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !strings.Contains(buf.String(), "<!DOCTYPE html>") {
		t.Error("expected HTML output")
	}
}

func TestNewFromBytes_ReportingMode(t *testing.T) {
	a, err := NewFromBytes([]byte(testRulesYAML), "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{
		Summary: types.Summary{NumDiffCRs: 1, TotalCRs: 5},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  config: expected\n+  config: found",
				CorrelatedTemplate: "test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
			},
		},
	}

	var buf bytes.Buffer
	if err := a.Analyze(&buf, report, "text", "reporting"); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if buf.String() == "" {
		t.Error("expected non-empty output for reporting mode")
	}
}

func TestAnalyze_WithMissingCRs(t *testing.T) {
	rulesFile := createTestRulesFile(t)
	a, err := New(rulesFile, "")
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 2,
			NumDiffCRs: 0,
			TotalCRs:   10,
			ValidationIssues: types.ValidationIssues{
				"required-config": {
					"missing-cr": types.Deviation{
						Msg: "Missing required CR",
						CRs: []string{"required/TestCR.yaml"},
					},
				},
			},
		},
		Diffs: []types.Diff{},
	}

	var buf bytes.Buffer
	err = a.Analyze(&buf, report, "text", "simple")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty output")
	}
}
