package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openshift-kni/rds-analyzer/internal/rules"
	"github.com/openshift-kni/rds-analyzer/internal/types"
)

const testHTMLRulesYAML = `
version: "1.0"
description: "Test Rules for HTML Generator"

settings:
  default_impact: "NeedsReview"
  default_severity: "MEDIUM"

label_annotation_rules:
  labels: []
  annotations: []
  default_impact: "NotADeviation"
  default_comment: "Labels and annotations are acceptable"

count_rules:
  - id: "C001-test"
    description: "Test count rule"
    match:
      templateFileName: "TestCR.yaml"
      crName: "*"
    limits:
      - condition: "count > 1"
        impact: "Impacting"
        comment: "Too many CRs"

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
  - id: "R002-impacting"
    description: "Impacting test rule"
    match:
      crName: "*"
    conditions:
      - type: "FoundNotExpected"
        contains: "dangerous:"
        impact: "Impacting"
        comment: "Dangerous configuration found"
`

func createHTMLTestRulesFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "rules.yaml")
	if err := os.WriteFile(rulesFile, []byte(testHTMLRulesYAML), 0644); err != nil {
		t.Fatalf("Failed to create test rules file: %v", err)
	}
	return rulesFile
}

func TestNewHTMLGenerator(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	if generator == nil {
		t.Fatal("expected generator, got nil")
	}
	if generator.ruleEngine != engine {
		t.Error("generator should have the rule engine set")
	}
	if generator.tmpl == nil {
		t.Error("generator should have template set")
	}
}

func TestHTMLGenerator_Generate_EmptyReport(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(output, "<html") {
		t.Error("expected html tag")
	}
}

func TestHTMLGenerator_Generate_WithDiffs(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngineWithVersion(rulesFile, "4.20")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 0,
			NumDiffCRs: 2,
			TotalCRs:   10,
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  name: expected\n+  name: found",
				CorrelatedTemplate: "required/test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
				Description:        "Test ConfigMap",
			},
			{
				DiffOutput:         "+  dangerous: true",
				CorrelatedTemplate: "required/test/DangerCR.yaml",
				CRName:             "v1_ConfigMap_default_danger",
				Description:        "Dangerous ConfigMap",
			},
		},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()

	// Verify HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}

	// Verify OCP version appears
	if !strings.Contains(output, "4.20") {
		t.Error("expected OCP version in output")
	}

	// Verify diff content appears
	if !strings.Contains(output, "ConfigMap") {
		t.Error("expected CR type in output")
	}
}

func TestHTMLGenerator_Generate_WithMissingCRs(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 2,
			NumDiffCRs: 0,
			TotalCRs:   10,
			ValidationIssues: types.ValidationIssues{
				"required-config": {
					"missing-sriov": types.Deviation{
						Msg: "Missing required SRIOV configuration",
						CRs: []string{"required/sriov/SriovConfig.yaml"},
					},
				},
				"optional-ptp": {
					"missing-ptp": types.Deviation{
						Msg: "Missing optional PTP configuration",
						CRs: []string{"optional/ptp/PtpConfig.yaml"},
					},
				},
			},
		},
		Diffs: []types.Diff{},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
}

func TestHTMLGenerator_Generate_WithUnmatchedCRs(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 0,
			NumDiffCRs: 0,
			TotalCRs:   5,
			UnmatchedCRS: []string{
				"v1_ConfigMap_default_unknown",
				"v1_Secret_default_mystery",
			},
		},
		Diffs: []types.Diff{},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "ampersand",
			input:    "foo & bar",
			expected: "foo &amp; bar",
		},
		{
			name:     "less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "multiple special characters",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert('xss')&lt;/script&gt;",
		},
		{
			name:     "yaml content",
			input:    "key: value & more",
			expected: "key: value &amp; more",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeHTML(tt.input)
			if string(result) != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHTMLGenerator_ImpactCSSClasses(t *testing.T) {
	// Test that HTML output contains correct CSS classes for different impacts
	// This indirectly tests the getImpactCSS function through the generated HTML
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumDiffCRs: 1,
			TotalCRs:   5,
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "+  extra: field",
				CorrelatedTemplate: "required/test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
			},
		},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()
	// Verify that CSS classes are defined in the output
	if !strings.Contains(output, "impacting") && !strings.Contains(output, "not-impacting") &&
		!strings.Contains(output, "needs-review") {
		t.Error("expected impact CSS classes in HTML output")
	}
}

func TestHTMLGenerator_Generate_FullReport(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngineWithVersion(rulesFile, "4.19")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing:   1,
			NumDiffCRs:   2,
			TotalCRs:     15,
			MetadataHash: "abc123",
			PatchedCRs:   3,
			UnmatchedCRS: []string{"v1_ConfigMap_default_extra"},
			ValidationIssues: types.ValidationIssues{
				"required-config": {
					"missing-cr": types.Deviation{
						Msg: "Missing required CR",
						CRs: []string{"required/TestCR.yaml"},
					},
				},
			},
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "-  expected: value\n+  found: different",
				CorrelatedTemplate: "required/test/ConfigA.yaml",
				CRName:             "v1_ConfigMap_ns_configA",
				Description:        "Config A",
			},
			{
				DiffOutput:         "+  extra: field",
				CorrelatedTemplate: "optional/test/ConfigB.yaml",
				CRName:             "v1_ConfigMap_ns_configB",
				Description:        "Config B",
			},
		},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()

	// Verify basic HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(output, "</html>") {
		t.Error("missing closing html tag")
	}

	// Verify CSS is included
	if !strings.Contains(output, "<style>") {
		t.Error("missing style tag")
	}
}

func TestHTMLGenerator_ProcessMissingCRs_Empty(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	groups, stats := generator.processMissingCRs(nil)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for nil issues, got %d", len(groups))
	}
	if stats.RequiredCRCount != 0 || stats.OptionalCRCount != 0 {
		t.Error("expected zero counts for empty issues")
	}
}

func TestHTMLGenerator_ProcessMissingCRs_WithData(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	issues := types.ValidationIssues{
		"required-sriov": {
			"sriov-config": types.Deviation{
				Msg: "Missing SRIOV",
				CRs: []string{"required/sriov/Config.yaml"},
			},
		},
		"optional-ptp": {
			"ptp-config": types.Deviation{
				Msg: "Missing PTP",
				CRs: []string{"optional/ptp/Config.yaml"},
			},
		},
	}

	groups, stats := generator.processMissingCRs(issues)

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}

	// Check that required group is marked correctly
	foundRequired := false
	foundOptional := false
	for _, g := range groups {
		if strings.HasPrefix(g.GroupName, "required-") && g.IsRequired {
			foundRequired = true
		}
		if strings.HasPrefix(g.GroupName, "optional-") && !g.IsRequired {
			foundOptional = true
		}
	}
	if !foundRequired {
		t.Error("expected to find required group marked as IsRequired")
	}
	if !foundOptional {
		t.Error("expected to find optional group marked as not IsRequired")
	}

	// Verify stats are populated
	totalMissing := stats.RequiredCRCount + stats.OptionalCRCount
	if totalMissing == 0 {
		t.Error("expected non-zero missing CR count")
	}
}

func TestHTMLGenerator_ValueDiffs(t *testing.T) {
	rulesFile := createHTMLTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumDiffCRs: 1,
			TotalCRs:   5,
		},
		Diffs: []types.Diff{
			{
				DiffOutput: ` spec:
   replicas:
-    count: 3
+    count: 5
   strategy:`,
				CorrelatedTemplate: "required/Deployment.yaml",
				CRName:             "apps/v1_Deployment_default_myapp",
				Description:        "Deployment with replica change",
			},
		},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML output")
	}
}

func TestHTMLGenerator_AllImpactCSSClasses(t *testing.T) {
	// Create rules that produce different impacts
	rulesYAML := `
version: "1.0"
description: "Rules for testing all impact CSS classes"

settings:
  default_impact: "NeedsReview"

label_annotation_rules:
  labels: []
  annotations: []
  default_impact: "NotADeviation"
  default_comment: "Labels OK"

rules:
  - id: "R001-impacting"
    description: "Impacting rule"
    match:
      crName: "*impacting*"
    conditions:
      - type: "Any"
        contains: "bad"
        impact: "Impacting"
        comment: "Bad config"
  - id: "R002-not-impacting"
    description: "Not impacting rule"
    match:
      crName: "*notimpacting*"
    conditions:
      - type: "Any"
        contains: "warning"
        impact: "NotImpacting"
        comment: "Warning only"
  - id: "R003-not-deviation"
    description: "Not a deviation"
    match:
      crName: "*notdeviation*"
    conditions:
      - type: "Any"
        contains: "ok"
        impact: "NotADeviation"
        comment: "This is fine"
`
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "rules.yaml")
	if err := os.WriteFile(rulesFile, []byte(rulesYAML), 0644); err != nil {
		t.Fatalf("Failed to write rules: %v", err)
	}

	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewHTMLGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumDiffCRs: 3,
			TotalCRs:   10,
		},
		Diffs: []types.Diff{
			{
				DiffOutput:         "+  bad: config",
				CorrelatedTemplate: "test/impacting.yaml",
				CRName:             "v1_ConfigMap_default_impacting",
			},
			{
				DiffOutput:         "+  warning: level",
				CorrelatedTemplate: "test/notimpacting.yaml",
				CRName:             "v1_ConfigMap_default_notimpacting",
			},
			{
				DiffOutput:         "+  ok: value",
				CorrelatedTemplate: "test/notdeviation.yaml",
				CRName:             "v1_ConfigMap_default_notdeviation",
			},
		},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()

	// Verify all CSS classes are present in the output
	cssClasses := []string{
		"impact-impacting",
		"impact-not-impacting",
		"impact-not-deviation",
		"impact-needs-review",
	}

	for _, class := range cssClasses {
		if !strings.Contains(output, class) {
			t.Errorf("expected CSS class %q in output", class)
		}
	}
}
