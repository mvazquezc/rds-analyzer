package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openshift-kni/rds-analyzer/internal/parser"
	"github.com/openshift-kni/rds-analyzer/internal/rules"
	"github.com/openshift-kni/rds-analyzer/internal/types"
)

const testTextRulesYAML = `
version: "1.0"
description: "Test Rules for Text Generator"

settings:
  default_impact: "NeedsReview"
  default_severity: "MEDIUM"

label_annotation_rules:
  impacting_labels: []
  impacting_annotations: []
  default_impact: "NotADeviation"
  default_comment: "Labels and annotations are acceptable"

count_rules:
  - id: "C001-test-count"
    description: "Test count rule"
    match:
      templateFileName: "TestCR.yaml"
      crName: "*_ConfigMap_*"
    limits:
      - condition: "count > 1"
        impact: "Impacting"
        comment: "Found {count} CRs, expected only 1"
        supporting_doc: "https://docs.example.com/count-rules"

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

func createTextTestRulesFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "rules.yaml")
	if err := os.WriteFile(rulesFile, []byte(testTextRulesYAML), 0644); err != nil {
		t.Fatalf("Failed to create test rules file: %v", err)
	}
	return rulesFile
}

func TestTextGenerator_ContextualDiffView(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngineWithVersion(rulesFile, "4.20")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)

	// Create a report with a diff that has contextual views.
	report := types.ValidationReport{
		Summary: types.Summary{
			ValidationIssues: types.ValidationIssues{},
			NumDiffCRs:       1,
			TotalCRs:         1,
		},
		Diffs: []types.Diff{
			{
				DiffOutput: ` spec:
   pipelines:
-    name: audit
+    name: infrastructure
   outputRefs:`,
				CorrelatedTemplate: "required/test/TestCR.yaml",
				CRName:             "v1_ConfigMap_default_test",
				Description:        "Test CR",
			},
		},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()

	// Verify expected and found sections appear.
	if !strings.Contains(output, "expected:") {
		t.Error("Missing 'expected:' section")
	}
	if !strings.Contains(output, "found:") {
		t.Error("Missing 'found:' section")
	}

	// Verify context lines appear (in dim color).
	if !strings.Contains(output, "spec:") {
		t.Error("Missing context line 'spec:'")
	}
	if !strings.Contains(output, "pipelines:") {
		t.Error("Missing context line 'pipelines:'")
	}

	// Verify changed lines appear with colors.
	if !strings.Contains(output, parser.ColorGreen) {
		t.Error("Missing green color for expected lines")
	}
	if !strings.Contains(output, parser.ColorRed) {
		t.Error("Missing red color for found lines")
	}
	if !strings.Contains(output, parser.ColorDim) {
		t.Error("Missing dim color for context lines")
	}
}

func TestTextGenerator_printContextualDiffViewColored(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)

	diffLines := []types.DiffLine{
		{Content: "  context1", IsChanged: false},
		{Content: "  changed_line", IsChanged: true},
		{Content: "  context2", IsChanged: false},
	}

	var buf bytes.Buffer
	generator.writer = &buf
	generator.printContextualDiffViewColored(diffLines, parser.ColorGreen)

	output := buf.String()

	// Verify context lines are dim.
	if !strings.Contains(output, parser.ColorDim+"  context1"+parser.ColorReset) {
		t.Error("Context line 1 not properly formatted with dim color")
	}
	if !strings.Contains(output, parser.ColorDim+"  context2"+parser.ColorReset) {
		t.Error("Context line 2 not properly formatted with dim color")
	}

	// Verify changed line has the specified color.
	if !strings.Contains(output, parser.ColorGreen+"  changed_line"+parser.ColorReset) {
		t.Error("Changed line not properly formatted with green color")
	}
}

func TestGetImpactSymbol(t *testing.T) {
	tests := []struct {
		impact   string
		expected string
	}{
		{"Impacting", "\U0001F534"},     // Red circle
		{"NotImpacting", "\U0001F7E1"},  // Yellow circle
		{"NotADeviation", "\U0001F7E2"}, // Green circle
		{"NeedsReview", "\u26AA"},       // White circle (default)
		{"Unknown", "\u26AA"},           // White circle (default)
	}

	for _, tt := range tests {
		t.Run(tt.impact, func(t *testing.T) {
			result := getImpactSymbol(tt.impact)
			if result != tt.expected {
				t.Errorf("getImpactSymbol(%q) = %q, want %q", tt.impact, result, tt.expected)
			}
		})
	}
}

func TestGetImpactColor(t *testing.T) {
	tests := []struct {
		impact   string
		expected string
	}{
		{"Impacting", parser.ColorRed + parser.ColorBold},
		{"NotImpacting", parser.ColorYellow},
		{"NotADeviation", parser.ColorGreen},
		{"NeedsReview", parser.ColorCyan},
		{"Unknown", parser.ColorCyan},
	}

	for _, tt := range tests {
		t.Run(tt.impact, func(t *testing.T) {
			result := getImpactColor(tt.impact)
			if result != tt.expected {
				t.Errorf("getImpactColor(%q) = %q, want %q", tt.impact, result, tt.expected)
			}
		})
	}
}

func TestTextGenerator_Generate_WithMissingCRs(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
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
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "MISSING CUSTOM RESOURCES") {
		t.Error("expected missing CRs section header")
	}
	if !strings.Contains(output, "required-config") {
		t.Error("expected group name in output")
	}
}

func TestTextGenerator_Generate_NoMissingCRs(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumMissing: 0,
			NumDiffCRs: 0,
			TotalCRs:   5,
		},
		Diffs: []types.Diff{},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No missing CRs found") {
		t.Error("expected 'No missing CRs found' message")
	}
}

func TestTextGenerator_Generate_NoDiffs(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	report := types.ValidationReport{
		Summary: types.Summary{
			NumDiffCRs: 0,
			TotalCRs:   5,
		},
		Diffs: []types.Diff{},
	}

	var buf bytes.Buffer
	err = generator.Generate(&buf, report)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No differences detected") {
		t.Error("expected 'No differences detected' message")
	}
}

func TestTextGenerator_DetermineImpact(t *testing.T) {
	tests := []struct {
		name           string
		matched        bool
		impact         string
		hasNeedsReview bool
		expected       string
	}{
		{
			name:           "impacting takes priority",
			matched:        true,
			impact:         "Impacting",
			hasNeedsReview: false,
			expected:       "Impacting",
		},
		{
			name:           "not impacting when matched",
			matched:        true,
			impact:         "NotImpacting",
			hasNeedsReview: false,
			expected:       "NotImpacting",
		},
		{
			name:           "not a deviation only",
			matched:        true,
			impact:         "NotADeviation",
			hasNeedsReview: false,
			expected:       "NotADeviation",
		},
		{
			name:           "needs review flag overrides non-impacting",
			matched:        true,
			impact:         "NotADeviation",
			hasNeedsReview: true,
			expected:       "NeedsReview",
		},
		{
			name:           "unmatched returns needs review",
			matched:        false,
			impact:         "",
			hasNeedsReview: false,
			expected:       "NeedsReview",
		},
		{
			name:           "impacting not overridden by needs review flag",
			matched:        true,
			impact:         "Impacting",
			hasNeedsReview: true,
			expected:       "Impacting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rules.EvaluationResult{
				Matched: tt.matched,
				Impact:  tt.impact,
			}
			impact := determineImpact(result, tt.hasNeedsReview)
			if impact != tt.expected {
				t.Errorf("determineImpact() = %q, want %q", impact, tt.expected)
			}
		})
	}
}

func TestTextGenerator_PrintDiffCheck_ExpectedNotFound(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	var buf bytes.Buffer
	generator.writer = &buf

	diffCheck := types.DiffCheck{
		CRName:           "v1_ConfigMap_default_test",
		TemplateFileName: "TestCR.yaml",
		ExpectedNotFound: []string{"  key: value"},
	}

	ruleResult := rules.EvaluationResult{
		Matched: false,
		Impact:  "NeedsReview",
	}

	impact := generator.printDiffCheck(diffCheck, ruleResult)

	output := buf.String()
	if !strings.Contains(output, "expected but not found:") {
		t.Error("expected 'expected but not found:' section")
	}
	if impact == "" {
		t.Error("expected non-empty impact")
	}
}

func TestTextGenerator_PrintDiffCheck_FoundNotExpected(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	var buf bytes.Buffer
	generator.writer = &buf

	diffCheck := types.DiffCheck{
		CRName:           "v1_ConfigMap_default_test",
		TemplateFileName: "TestCR.yaml",
		FoundNotExpected: []string{"  extra: field"},
	}

	ruleResult := rules.EvaluationResult{
		Matched: false,
		Impact:  "NeedsReview",
	}

	impact := generator.printDiffCheck(diffCheck, ruleResult)

	output := buf.String()
	if !strings.Contains(output, "found but not expected:") {
		t.Error("expected 'found but not expected:' section")
	}
	if impact == "" {
		t.Error("expected non-empty impact")
	}
}

func TestTextGenerator_PrintDiffCheck_ValueDiff(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	var buf bytes.Buffer
	generator.writer = &buf

	diffCheck := types.DiffCheck{
		CRName:           "v1_ConfigMap_default_test",
		TemplateFileName: "TestCR.yaml",
		ExpectedValue:    []string{"  name: expected"},
		FoundValue:       []string{"  name: found"},
	}

	ruleResult := rules.EvaluationResult{
		Matched: true,
		RuleID:  "R001-test",
		Impact:  "NotImpacting",
		Conditions: []rules.ConditionResult{
			{
				RuleID:        "R001-test",
				ConditionType: "ExpectedFound",
				Matched:       true,
				Impact:        "NotImpacting",
				MatchedText:   "  name: found",
			},
		},
	}

	impact := generator.printDiffCheck(diffCheck, ruleResult)

	output := buf.String()
	if !strings.Contains(output, "expected:") {
		t.Error("expected 'expected:' section")
	}
	if !strings.Contains(output, "found:") {
		t.Error("expected 'found:' section")
	}
	if impact != "NotImpacting" {
		t.Errorf("expected NotImpacting impact, got %s", impact)
	}
}

func TestTextGenerator_PrintCountRuleResults(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	var buf bytes.Buffer
	generator.writer = &buf

	countResults := []rules.CountRuleResult{
		{
			RuleID:      "C001-test",
			Description: "Test count rule violation",
			Matched:     true,
			Count:       3,
			Impact:      "Impacting",
			Comment:     "Found 3 CRs, expected only 1",
			MatchedCRs: []string{
				"v1_ConfigMap_ns1_config1",
				"v1_ConfigMap_ns2_config2",
				"v1_ConfigMap_ns3_config3",
			},
		},
	}

	impactStats := make(map[string]int)
	generator.printCountRuleResults(countResults, impactStats)

	output := buf.String()

	// Verify section header
	if !strings.Contains(output, "COUNT RULE VIOLATIONS") {
		t.Error("expected COUNT RULE VIOLATIONS header")
	}

	// Verify rule details
	if !strings.Contains(output, "C001-test") {
		t.Error("expected rule ID in output")
	}
	if !strings.Contains(output, "Test count rule violation") {
		t.Error("expected description in output")
	}
	if !strings.Contains(output, "3 CRs matched") {
		t.Error("expected count in output")
	}
	if !strings.Contains(output, "Impacting") {
		t.Error("expected impact in output")
	}
	if !strings.Contains(output, "Found 3 CRs") {
		t.Error("expected comment in output")
	}

	// Verify matched CRs are listed
	if !strings.Contains(output, "v1_ConfigMap_ns1_config1") {
		t.Error("expected matched CR in output")
	}

	// Verify impact stats updated
	if impactStats["Impacting"] != 1 {
		t.Errorf("expected impactStats[Impacting] = 1, got %d", impactStats["Impacting"])
	}
}

func TestTextGenerator_PrintCountRuleResults_Empty(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	var buf bytes.Buffer
	generator.writer = &buf

	impactStats := make(map[string]int)
	generator.printCountRuleResults([]rules.CountRuleResult{}, impactStats)

	output := buf.String()

	// Should still print header
	if !strings.Contains(output, "COUNT RULE VIOLATIONS") {
		t.Error("expected COUNT RULE VIOLATIONS header")
	}

	// No impact stats should be updated
	if len(impactStats) != 0 {
		t.Errorf("expected empty impactStats, got %v", impactStats)
	}
}

func TestTextGenerator_PrintCountRuleResults_NoMatchedCRs(t *testing.T) {
	rulesFile := createTextTestRulesFile(t)
	engine, err := rules.NewEngine(rulesFile)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	generator := NewTextGenerator(engine)
	var buf bytes.Buffer
	generator.writer = &buf

	countResults := []rules.CountRuleResult{
		{
			RuleID:      "C002-empty",
			Description: "No CRs found",
			Matched:     true,
			Count:       0,
			Impact:      "Impacting",
			Comment:     "No CRs configured",
			MatchedCRs:  []string{},
		},
	}

	impactStats := make(map[string]int)
	generator.printCountRuleResults(countResults, impactStats)

	output := buf.String()

	// Verify rule is printed
	if !strings.Contains(output, "C002-empty") {
		t.Error("expected rule ID in output")
	}

	// Should NOT contain "Matched CRs:" since list is empty
	if strings.Contains(output, "Matched CRs:") {
		t.Error("should not print 'Matched CRs:' when list is empty")
	}
}
