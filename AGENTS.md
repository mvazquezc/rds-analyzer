# AI Agent Guidelines for RDS Analyzer

This document provides guidelines for AI agents working with the RDS Analyzer codebase.

## Project Overview

RDS Analyzer is a Go CLI tool that evaluates evaluates kube-compare JSON reports against a set of rules. It determines the impact of configuration deviations and generates text or HTML reports.

## Architecture

### Package Structure

```
internal/
├── analyzer/    # Orchestration - coordinates rule engine and report generation
├── cli/         # Cobra CLI - command parsing and flag handling
├── parser/      # Diff parsing - transforms unified diff to structured data
├── report/      # Output generators - text (terminal) and HTML formats
├── rules/       # Rule engine - pattern matching and impact resolution
└── types/       # Data structures - shared types for validation reports
```

### Data Flow

1. CLI reads JSON input (file or stdin)
2. JSON is parsed into `types.ValidationReport`
3. `analyzer.Analyzer` orchestrates processing:
   - Loads rules via `rules.Engine`
   - Passes report to appropriate generator
4. Report generator:
   - Uses `parser` to transform diffs
   - Uses `rules.Engine` to evaluate each diff
   - Outputs formatted results

## Code Conventions

### Go Style

- Follow standard Go conventions (`gofmt`, `goimports`)
- Use meaningful variable names; avoid single letters except for loops
- Prefer early returns over nested conditionals
- Keep functions focused and small

### Error Handling

- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return errors to callers; let main/CLI handle user-facing output
- Avoid `log.Fatal` except in `main.go`

### Comments

- Package-level comments explain purpose
- Exported functions have doc comments starting with function name
- Complex logic has inline comments explaining "why"

### Imports

```go
import (
    // Standard library first
    "fmt"
    "io"

    // External dependencies
    "github.com/spf13/cobra"

    // Internal packages
    "github.com/telco-operations/rds-analyzer/internal/types"
)
```

## Common Tasks

### Adding a New CLI Flag

1. Edit `internal/cli/root.go`
2. Add variable at package level
3. Register in `init()` with `rootCmd.Flags()`
4. Use in `runAnalysis()`

### Adding a New Rule Condition Type

1. Define in `internal/rules/types.go` (add to Condition.Type options)
2. Handle in `internal/rules/engine.go`:
   - Add case in `evaluateCondition()`
   - Implement matching logic
3. **Add tests in `internal/rules/engine_test.go`**:
   - Add test cases to `TestConditionTypes` for the new condition type
   - Add example rules using the new type to `testRulesYAML` constant
   - Add realistic scenarios to `TestEvaluate` or `TestEvaluateFromOutputJSON`

### Adding a New Rule Type (e.g., Count Rules, Label Rules)

1. Define types in `internal/rules/types.go`
2. Implement evaluation in `internal/rules/engine.go`
3. **Add comprehensive tests in `internal/rules/engine_test.go`**:
   - Create a dedicated test function (e.g., `TestEvaluateNewRuleType`)
   - Add the new rule type to `testRulesYAML` constant
   - Test edge cases (empty input, no match, multiple matches)
   - Test impact priority when combined with other rules

### Adding a New Output Format

1. Create `internal/report/newformat.go`
2. Implement generator with `Generate(io.Writer, types.ValidationReport) error`
3. Add case in `internal/analyzer/analyzer.go`
4. Add CLI option in `internal/cli/root.go`

### Modifying the HTML Template

The HTML template is embedded in `internal/report/html.go` as the `htmlTemplate` constant. Edit directly; Go will include it at build time.

## Testing Guidelines

### Unit Tests

- Place tests in `*_test.go` files alongside source
- Test public interfaces, not implementation details
- Use table-driven tests for multiple cases
- Name test functions `TestFunctionName_Scenario`

### Rules Engine Tests (`internal/rules/engine_test.go`)

The engine test file contains comprehensive tests for the rule evaluation system. When adding new rules or rule types, update this file:

**Key Test Functions:**
- `TestEvaluate` - Main evaluation with various diff scenarios
- `TestEvaluateFromOutputJSON` - Real scenarios from `testdata/output.json`
- `TestMatchesPattern` - Glob pattern matching
- `TestCheckContains` / `TestCheckRegex` - Pattern matching logic
- `TestEvaluateCountRules` - Count-based rule evaluation
- `TestEvaluateMissingCRs` - Missing CR impact (required/optional groups)
- `TestEvaluateLabelOrAnnotation` - Label/annotation evaluation
- `TestVersionedImpact` - Version-specific impact resolution
- `TestConditionTypes` - Different condition types (Any, FoundNotExpected, etc.)

**Adding Tests for New Rules:**

1. Update `testRulesYAML` constant with example rules using the new feature
2. Add test cases to appropriate test functions
3. Use realistic data derived from `testdata/output.json` when possible
4. Test both matching and non-matching scenarios
5. Test impact priority when multiple rules match

### Test Patterns

```go
func TestParseOCPVersion_Valid(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    OCPVersion
        wantErr bool
    }{
        {"standard", "4.19", OCPVersion{4, 19}, false},
        {"empty", "", OCPVersion{}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseOCPVersion(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Tests

```bash
make test           # Run all tests
make test-coverage  # Generate coverage report
```

## Rules Engine Details

### Pattern Matching

The engine supports:
- Exact match: `templateFileName: "exact.yaml"`
- Glob wildcard: `crName: "namespace/v1_*_openshift-*"`

### Impact Resolution

For versioned impacts:
1. Exact match uses that version's impact
2. No exact match: inherit from highest version <= target
3. Target below all versions: use lowest defined version

### Condition Evaluation

Order of precedence:
1. `regex` (if set, checked first)
2. `contains` (substring match)

The `Any` type checks all diff sections in order:
1. FoundNotExpected
2. ExpectedNotFound
3. FoundValue (from ExpectedFound)

## Build and Release

### Build Commands

```bash
make build          # Current platform
make build-all      # All platforms
make install        # Install to GOBIN
```

### Version Information

Set via ldflags at build time:
- `cli.Version`: Git tag or "dev"
- `cli.Commit`: Git short hash
- `cli.BuildDate`: Build timestamp

## Troubleshooting

### Common Issues

**"failed to read rules file"**
- Check rules.yaml path (-r flag)
- Default looks in current directory

**"invalid OCP version"**
- Format must be `Major.Minor` (e.g., "4.19")
- No patch version or prefixes

**Empty output**
- Verify input JSON structure matches `types.ValidationReport`
- Check for empty Diffs array

### Debugging

Add debug output to stderr (won't affect piped output):
```go
fmt.Fprintf(os.Stderr, "DEBUG: %v\n", value)
```

## Code Quality Checklist

Before submitting changes:

- [ ] `make fmt` - code formatted
- [ ] `make vet` - no issues
- [ ] `make lint` - linter passes
- [ ] `make test` - tests pass
- [ ] New code has comments
- [ ] Public functions have doc comments
- [ ] Errors are wrapped with context
- [ ] No hardcoded paths or values
- [ ] **New rule types have corresponding tests in `engine_test.go`**
- [ ] **Test coverage remains above 80%** (`go test -cover ./internal/rules/...`)

---

## Writing Rules

This section provides comprehensive guidance for writing validation rules in YAML. Rules define how configuration deviations are evaluated and what impact they have.

### Rule File Structure

A complete rules file has this top-level structure:

```yaml
version: "1.0"
description: "Description of this rule set"

settings:
  default_impact: "NeedsReview"      # Impact when no rule matches
  default_severity: "MEDIUM"         # Severity categorization

label_annotation_rules:              # Label/annotation validation
  labels: []
  annotations: []
  default_impact: "NotADeviation"
  default_comment: "..."

count_rules: []                      # Aggregate count checks
global_rules: []                     # Rules applying to all CRs
rules: []                            # Rules for specific CRs
```

### Rule ID Naming Convention

Rule IDs must follow this naming convention:

```
<PREFIX>-<TYPE>-<NUMBER>-<descriptive-name>
```

**Components:**
- `<PREFIX>`: Project or profile identifier (ask user for this)
- `<TYPE>`: Rule type indicator
  - `R` - Standard rules (specific CR matching)
  - `G` - Global rules (apply to all CRs)
  - `C` - Count rules (aggregate checks)
- `<NUMBER>`: Three-digit sequential number (001, 002, etc.)
- `<descriptive-name>`: Kebab-case description of what the rule validates

**Examples:**
```yaml
id: "RAN-R-001-network-diagnostics"      # Standard rule
id: "RAN-G-001-sysctls"                  # Global rule
id: "RAN-C-001-single-catalogsource"     # Count rule
id: "CORE-R-015-resource-limits"         # Different prefix
```

**When adding rules:** Ask the user for the appropriate prefix if not already established in the rules file.

### Impact Levels

Impact levels indicate the severity of a deviation. Listed from most to least severe:

| Impact | Description | Use When |
|--------|-------------|----------|
| `Impacting` | Critical issue affecting functionality or compliance | Configuration breaks functionality, violates requirements, or causes performance/stability issues |
| `NeedsReview` | Requires manual evaluation | Impact is context-dependent or requires human judgment |
| `NotImpacting` | Found deviation but doesn't affect system | Deviation is understood and has no negative effect |
| `NotADeviation` | Expected difference, not a problem | Intentional customization or optional field |

**Impact Resolution Priority:** When multiple conditions match, the engine returns the worst (most severe) impact. Priority order: `Impacting` > `NeedsReview` > `NotImpacting` > `NotADeviation`.

### Versioned Impacts

Impacts can vary by OCP version. Use versioned impacts when a deviation's severity changes across versions:

```yaml
# Simple impact (same for all versions)
impact: "Impacting"

# Versioned impact (varies by OCP version)
impact:
  4.18: NotImpacting
  4.19: NeedsReview
  4.20: Impacting
```

**Version Resolution Logic:**
1. Exact version match uses that version's impact
2. No exact match: inherits from highest version ≤ target
3. Target below all defined versions: uses lowest defined version

**Example Resolution:**
```yaml
impact:
  4.18: NotImpacting
  4.20: Impacting
```
- OCP 4.18 → `NotImpacting` (exact match)
- OCP 4.19 → `NotImpacting` (inherits from 4.18)
- OCP 4.20 → `Impacting` (exact match)
- OCP 4.21 → `Impacting` (inherits from 4.20)
- OCP 4.17 → `NotImpacting` (below all, uses lowest)

### Standard Rules

Standard rules match specific CRs and evaluate conditions against their diffs.

#### Basic Structure

```yaml
rules:
  - id: "PREFIX-R-001-descriptive-name"
    description: "Human-readable explanation of what this rule validates"
    match:
      templateFileName: "TemplateName.yaml"
      crName: "group/version_Kind_namespace_name"
    conditions:
      - type: "ExpectedFound"
        contains: "pattern to match"
        impact:
          4.20: Impacting
        comment: "Explanation and guidance for this condition"
        supporting_doc: "https://docs.example.com/relevant-page"
```

#### Match Patterns

The `match` field determines which CRs this rule applies to. Both fields support glob-style wildcards (`*`).

```yaml
match:
  templateFileName: "DisableSnoNetworkDiag.yaml"    # Exact match
  crName: "operator.openshift.io/v1_Network_cluster"
```

**Template Filename Patterns:**
```yaml
templateFileName: "DisableSnoNetworkDiag.yaml"     # Exact match
templateFileName: "06-kdump-*.yaml"                # Glob: 06-kdump-enable-worker.yaml, etc.
templateFileName: "SriovNetworkNodePolicy*.yaml"   # Glob: multiple policies
templateFileName: ""                               # Empty: matches all templates
```

**CR Name Patterns:**

CR names follow the format: `group/version_Kind_namespace_name` or `version_Kind_name` for cluster-scoped resources.

```yaml
crName: "operator.openshift.io/v1_Network_cluster"                           # Exact match
crName: "operators.coreos.com/v1alpha1_CatalogSource_openshift-marketplace_*" # Glob on name
crName: "v1_Namespace_openshift-*"                                           # Glob: openshift namespaces
crName: "machineconfiguration.openshift.io/v1_MachineConfig_*"               # All MachineConfigs
```

**Omitting Match Fields:**
- Omit `templateFileName` to match any template
- Omit `crName` to match any CR (but must have at least one field or use global_rules)
- Use `match: {}` only in global_rules

#### Condition Types

The `type` field specifies which diff section to check:

| Type | Checks | Use Case |
|------|--------|----------|
| `FoundNotExpected` | Lines in actual CR but not in template | Extra/unexpected configuration |
| `ExpectedNotFound` | Lines in template but not in actual CR | Missing required configuration |
| `ExpectedFound` | Lines where both exist but values differ | Value mismatches |
| `Any` | All three sections above | General pattern detection |

**How Diff Sections Work:**

When comparing actual CR against template:
- **FoundNotExpected**: Customer added something not in the reference
- **ExpectedNotFound**: Customer is missing something from the reference
- **ExpectedFound**: Both have the field but with different values (checks the actual value)

```yaml
conditions:
  # Detect missing required configuration
  - type: "ExpectedNotFound"
    contains: "workload.openshift.io/allowed: management"
    impact:
      4.20: Impacting
    comment: "Missing required workload management label"

  # Detect incorrect value
  - type: "ExpectedFound"
    contains: "disableNetworkDiagnostics: false"
    impact:
      4.20: Impacting
    comment: "Network diagnostics should be disabled (set to true)"

  # Detect extra configuration (may be problematic)
  - type: "FoundNotExpected"
    contains: "unsupported-feature: enabled"
    impact:
      4.20: Impacting
    comment: "This feature is not supported in the reference configuration"

  # Detect pattern anywhere in diff
  - type: "Any"
    regex: 'net\..*\..*'
    impact:
      4.20: NotImpacting
    comment: "Network sysctl detected - review for appropriateness"
```

#### Pattern Matching: Contains vs Regex

**Contains (Substring Match):**

Simple substring matching. Use for exact text patterns.

```yaml
# Single line
contains: "disableNetworkDiagnostics: false"

# Multi-line (all lines must appear in sequence)
contains: |
  spec:
    disableAllDefaultSources: true
```

**Regex (Regular Expression):**

Full regex syntax. Takes precedence over `contains` if both specified.

```yaml
# Match network sysctls
regex: 'net\..*\..*'

# Match memory values with ranges
regex: '.*crashkernel=(?:[1-9]\d*G|676M|67[6-9]M).*'

# Match version patterns
regex: 'version:\s*v?[0-9]+\.[0-9]+\.[0-9]+'
```

**When to Use Each:**

| Use `contains` when... | Use `regex` when... |
|------------------------|---------------------|
| Matching exact text | Matching patterns with variations |
| Simple field: value pairs | Numeric ranges or thresholds |
| Multi-line exact sequences | Multiple alternative patterns |
| Performance matters (faster) | Complex pattern logic needed |

#### Supporting Documentation

Always include `supporting_doc` when relevant documentation exists:

```yaml
conditions:
  - type: "Any"
    regex: 'net\..*\..*'
    impact:
      4.20: NotImpacting
    comment: "Sysctls beginning with net.* are network namespaced."
    supporting_doc: https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html/scalability_and_performance/telco-ran-du-ref-design-specs
```

#### Complete Standard Rule Example

```yaml
rules:
  - id: "RAN-R-003-kdump-memory"
    description: "Validates kdump crashkernel memory configuration meets minimum requirements"
    match:
      templateFileName: "06-kdump-*.yaml"
      crName: "machineconfiguration.openshift.io/v1_MachineConfig_06-kdump-enable-*"
    conditions:
      # Memory below minimum threshold - critical issue
      - type: "ExpectedFound"
        regex: '.*crashkernel=(?:[1-9]\d?|[1-5]\d\d|6[0-6]\d|67[0-5])M$'
        impact:
          4.20: Impacting
        comment: "Crashkernel memory must be set to 676M or higher for reliable kdump operation."
        supporting_doc: https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html/support/support-overview

      # Memory at or above threshold - acceptable
      - type: "ExpectedFound"
        regex: '.*crashkernel=(?:[1-9]\d*G|676M|67[6-9]M|6[8-9]\dM|[7-9]\d\dM|\d{4,}M)$'
        impact:
          4.20: NotImpacting
        comment: "Crashkernel memory is at or above the 676M minimum threshold."
```

### Global Rules

Global rules apply to all CRs regardless of template or CR name. Use for patterns that should be checked everywhere.

```yaml
global_rules:
  - id: "RAN-G-001-sysctls"
    description: "Detect and evaluate sysctl configurations across all CRs"
    match: {}    # Empty match - applies to everything
    conditions:
      - type: "Any"
        regex: 'net\..*\..*'
        impact:
          4.20: NotImpacting
        comment: "Network-namespaced sysctls detected. Review for resource impact."
        supporting_doc: https://docs.example.com/sysctls

      - type: "Any"
        contains: "kernel.unknown_nmi_panic"
        impact:
          4.20: Impacting
        comment: "This sysctl causes system panic on NMI, leading to extended outage."
```

**When to Use Global Rules:**
- Detecting patterns that can appear in any CR (sysctls, labels, annotations)
- Enforcing organization-wide policies
- Catching common misconfigurations

**Important:** Global rules are evaluated for every CR, so keep conditions efficient. Prefer specific rules when the pattern only applies to certain CRs.

### Count Rules

Count rules validate the number of CRs matching a pattern. Use for cardinality checks.

```yaml
count_rules:
  - id: "RAN-C-001-single-catalogsource"
    description: "Validates that exactly one CatalogSource is configured"
    match:
      templateFileName: "DefaultCatsrc.yaml"
      crName: "operators.coreos.com/v1alpha1_CatalogSource_openshift-marketplace_*"
    limits:
      - condition: "count > 1"
        impact:
          4.20: Impacting
        comment: "Found {count} CatalogSource CRs. Only one is supported."

      - condition: "count == 0"
        impact:
          4.20: Impacting
        comment: "No CatalogSource configured. At least one is required."
```

#### Count Condition Operators

| Operator | Example | Description |
|----------|---------|-------------|
| `>` | `count > 1` | More than N |
| `>=` | `count >= 2` | At least N |
| `<` | `count < 1` | Fewer than N |
| `<=` | `count <= 3` | At most N |
| `==` | `count == 0` | Exactly N |
| `!=` | `count != 1` | Not exactly N |

#### The `{count}` Placeholder

Use `{count}` in comments to include the actual count:

```yaml
comment: "Found {count} instances, expected exactly 1."
# Renders as: "Found 3 instances, expected exactly 1."
```

#### Count Rule Use Cases

```yaml
count_rules:
  # Ensure singleton resources
  - id: "PREFIX-C-001-single-instance"
    description: "Only one instance allowed"
    match:
      crName: "example.io/v1_Singleton_*"
    limits:
      - condition: "count > 1"
        impact:
          4.20: Impacting
        comment: "Multiple instances found ({count}). Only one is supported."

  # Ensure minimum replicas
  - id: "PREFIX-C-002-minimum-replicas"
    description: "At least 3 replicas required for HA"
    match:
      crName: "apps/v1_Deployment_critical-namespace_*"
    limits:
      - condition: "count < 3"
        impact:
          4.20: Impacting
        comment: "Only {count} replicas configured. Minimum 3 required for HA."

  # Warn on high count
  - id: "PREFIX-C-003-too-many-configmaps"
    description: "Excessive ConfigMaps may indicate configuration sprawl"
    match:
      crName: "v1_ConfigMap_*"
    limits:
      - condition: "count > 100"
        impact:
          4.20: NeedsReview
        comment: "Found {count} ConfigMaps. Review for consolidation opportunities."
```

### Label and Annotation Rules

Label and annotation rules evaluate metadata on CRs. They use specificity-based matching.

```yaml
label_annotation_rules:
  labels:
    - key: "openshift.io/cluster-monitoring"
      value: "true"
      description: "Cluster monitoring label deviation from reference"
      impact:
        4.20: Impacting

    - key: "app.kubernetes.io/*"
      description: "Application labels should follow Kubernetes conventions"
      impact:
        4.20: NotADeviation

  annotations:
    - key: "target.workload.openshift.io/management"
      description: "Workload management annotation required for proper scheduling"
      impact:
        4.20: Impacting

  default_impact: "NotADeviation"
  default_comment: "Labels and annotations not matching any rule are acceptable"
```

#### Label/Annotation Rule Fields

| Field | Required | Description |
|-------|----------|-------------|
| `key` | Yes | Key pattern (supports glob wildcards) |
| `value` | No | Value pattern (supports glob). If omitted, matches any value |
| `description` | Yes | Explains the impact of this label/annotation |
| `impact` | Yes | Impact level (supports versioning) |

#### Pattern Matching for Keys and Values

```yaml
labels:
  # Exact key and value
  - key: "openshift.io/cluster-monitoring"
    value: "true"
    description: "Exact match for monitoring label"
    impact:
      4.20: Impacting

  # Exact key, any value
  - key: "app.kubernetes.io/name"
    description: "Name label present (any value)"
    impact:
      4.20: NotADeviation

  # Glob key pattern
  - key: "custom.example.com/*"
    description: "Custom labels from example.com domain"
    impact:
      4.20: NeedsReview

  # Glob key and value
  - key: "tier-*"
    value: "prod*"
    description: "Production tier labels"
    impact:
      4.20: NotImpacting
```

#### Specificity Scoring

When multiple rules could match a label/annotation, the most specific rule wins:

| Pattern Type | Base Score |
|--------------|------------|
| Exact key + exact value | 600 |
| Exact key + glob value | 500 |
| Exact key + any value | 400 |
| Glob key + exact value | 300 |
| Glob key + glob value | 200 |
| Glob key + any value | 100 |

Bonus points are added for literal (non-wildcard) character length.

**Example:** For label `app.kubernetes.io/name: myapp`:
- Rule with `key: "app.kubernetes.io/name", value: "myapp"` → 600 + bonus (highest priority)
- Rule with `key: "app.kubernetes.io/*"` → 100 + bonus (lower priority)

### Multi-Line Pattern Matching

For complex configurations spanning multiple lines, use multi-line `contains`:

```yaml
conditions:
  - type: "ExpectedNotFound"
    contains: |
      spec:
        config:
          disableAllDefaultSources: true
    impact:
      4.20: Impacting
    comment: "Missing required multi-line configuration block"
```

**Behavior:**
- All lines in the pattern must appear in sequence
- Empty lines between matches are ignored
- Whitespace/indentation must match exactly

### Writing Effective Comments

Comments should provide actionable guidance. Include:

1. **What's wrong**: Describe the deviation
2. **Why it matters**: Explain the impact
3. **How to fix**: Provide remediation guidance (when appropriate)

**Good Comments:**
```yaml
comment: "Network diagnostics should be disabled for RAN deployments to reduce CPU overhead. Set disableNetworkDiagnostics: true."

comment: "Crashkernel memory of {value} is below the 676M minimum. Kdump may fail to capture crash dumps. Increase to at least 676M."

comment: "Missing workload.openshift.io/allowed: management label. This label is required for proper pod placement on reserved cores when using workload partitioning."
```

**Avoid:**
```yaml
# Too vague
comment: "This is wrong"

# No guidance
comment: "Value mismatch detected"

# Too verbose for simple issues
comment: "We have detected that the configuration value present in your cluster does not match..."
```

### Rule Evaluation Order

Understanding evaluation order helps write efficient rules:

1. **Global rules** are evaluated first (for every CR)
2. **Specific rules** are evaluated if `match` criteria apply
3. **All matching conditions** are collected
4. **Worst impact** wins (Impacting > NeedsReview > NotImpacting > NotADeviation)
5. **Duplicate conditions** are deduplicated

**Implication:** A global rule with `NotImpacting` won't override a specific rule with `Impacting` for the same pattern.

### Common Patterns and Recipes

#### Validate Required Fields

```yaml
- id: "PREFIX-R-001-required-annotation"
  description: "Validates required annotation is present"
  match:
    crName: "v1_Namespace_*"
  conditions:
    - type: "ExpectedNotFound"
      contains: "required-annotation-key"
      impact:
        4.20: Impacting
      comment: "Missing required annotation. Add 'required-annotation-key: value' to metadata.annotations."
```

#### Validate Value Ranges

```yaml
- id: "PREFIX-R-002-memory-limits"
  description: "Validates memory is within acceptable range"
  match:
    crName: "apps/v1_Deployment_*"
  conditions:
    # Too low
    - type: "ExpectedFound"
      regex: 'memory:\s*(?:[1-9]|[1-9]\d|[1-4]\d\d)Mi'
      impact:
        4.20: Impacting
      comment: "Memory limit below 500Mi minimum"

    # Acceptable range
    - type: "ExpectedFound"
      regex: 'memory:\s*(?:[5-9]\d\d|[1-9]\d{3,})Mi'
      impact:
        4.20: NotImpacting
      comment: "Memory limit is within acceptable range"
```

#### Allow Optional Differences

```yaml
- id: "PREFIX-R-003-optional-fields"
  description: "Mark optional field differences as non-deviations"
  match:
    crName: "example.io/v1_Config_*"
  conditions:
    - type: "ExpectedNotFound"
      contains: "displayName:"
      impact:
        4.20: NotADeviation
      comment: "displayName is optional and informational"

    - type: "ExpectedNotFound"
      contains: "description:"
      impact:
        4.20: NotADeviation
      comment: "description is optional and informational"

    - type: "ExpectedFound"
      contains: "description:"
      impact:
        4.20: NotADeviation
      comment: "description value differences are acceptable"
```

#### Detect Forbidden Configuration

```yaml
- id: "PREFIX-G-001-forbidden-config"
  description: "Detect configuration that should never be present"
  match: {}
  conditions:
    - type: "FoundNotExpected"
      contains: "privileged: true"
      impact:
        4.20: Impacting
      comment: "Privileged containers are not allowed. Remove privileged: true from securityContext."

    - type: "Any"
      regex: 'hostNetwork:\s*true'
      impact:
        4.20: NeedsReview
      comment: "Host networking detected. Verify this is required for the workload."
```

#### Version-Dependent Behavior

```yaml
- id: "PREFIX-R-004-feature-adoption"
  description: "Feature required in newer versions, optional in older"
  match:
    crName: "config.openshift.io/v1_FeatureGate_*"
  conditions:
    - type: "ExpectedNotFound"
      contains: "newFeature: enabled"
      impact:
        4.18: NotImpacting      # Optional in 4.18
        4.19: NeedsReview       # Recommended in 4.19
        4.20: Impacting         # Required in 4.20+
      comment: "newFeature is required starting in OCP 4.20"
```

### Testing Your Rules

After writing rules, verify they work correctly:

1. **Check YAML syntax:**
   ```bash
   python3 -c "import yaml; yaml.safe_load(open('rules.yaml'))"
   ```

2. **Run the analyzer with test data:**
   ```bash
   ./rds-analyzer -r rules.yaml -i testdata/output.json
   ```

3. **Test specific OCP versions:**
   ```bash
   ./rds-analyzer -r rules.yaml -i testdata/output.json --ocp-version 4.20
   ./rds-analyzer -r rules.yaml -i testdata/output.json --ocp-version 4.18
   ```

4. **Verify expected impacts:**
   - Check that critical issues show as `Impacting`
   - Check that acceptable variations show as `NotADeviation`
   - Check that ambiguous cases show as `NeedsReview`

### Checklist for New Rules

Before submitting new rules:

- [ ] Rule ID follows naming convention (`PREFIX-TYPE-NNN-description`)
- [ ] Description clearly explains what the rule validates
- [ ] Match patterns use appropriate specificity (not too broad, not too narrow)
- [ ] Condition types match the diff section being checked
- [ ] Regex patterns are tested and escaped properly
- [ ] Impacts are appropriate for the severity of the deviation
- [ ] Versioned impacts are used when behavior changes across OCP versions
- [ ] Comments provide actionable guidance
- [ ] `supporting_doc` is included when relevant documentation exists
- [ ] Rules are tested with real or realistic diff data
- [ ] No duplicate rules (check existing rules first)
