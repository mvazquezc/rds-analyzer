# RDS Analyzer

[![CI](https://github.com/openshift-kni/rds-analyzer/actions/workflows/ci.yml/badge.svg)](https://github.com/openshift-kni/rds-analyzer/actions/workflows/ci.yml)

A rule-based analyzer for OpenShift cluster comparisons. This tool evaluates [kube-compare](https://github.com/openshift/kube-compare) JSON reports against a configurable set of rules to determine the impact of configuration deviations from the reference configuration.

## Overview

- Evaluates configuration differences against YAML-defined rules.
- Determines impact levels: Impacting, Not Impacting, Not a Deviation, or Needs Review.
- Supports version-specific rule evaluation for different OCP versions.
- Generates text or HTML reports with detailed analysis.

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/openshift-kni/rds-analyzer.git
cd rds-analyzer

# Build
make build

# Or install directly to GOBIN
make install
```

### Binary Releases

Download pre-built binaries from the [Releases](https://github.com/openshift-kni/rds-analyzer/releases) page.

The built binary will be at `./build/rds-analyzer`.

### Container Image

Container images are available on Quay.io:

```bash
podman pull quay.io/rhsysdeseng/rds-analyzer:latest
```

## Usage

### Basic Usage

```bash
# Analyze from file with text output (default)
rds-analyzer -i results.json

# Analyze from stdin
cat results.json | rds-analyzer

# Generate HTML report
rds-analyzer -i results.json -o html > report.html

# Generate reporting format output
rds-analyzer -i results.json -m reporting
```

### Container Usage

```bash
# Analyze from file
podman run --rm -v $(pwd):/data:Z quay.io/rhsysdeseng/rds-analyzer:latest \
  -i /data/results.json -r /data/rules.yaml

# Generate HTML report
podman run --rm -v $(pwd):/data:Z quay.io/rhsysdeseng/rds-analyzer:latest \
  -i /data/results.json -r /data/rules.yaml -o html > report.html

# Analyze from stdin
cat results.json | podman run --rm -i -v $(pwd):/data:Z quay.io/rhsysdeseng/rds-analyzer:latest \
  -r /data/rules.yaml
```

### Command-Line Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--input` | `-i` | Input file path (reads from stdin if not specified) | stdin |
| `--output` | `-o` | Output format: `text` or `html` | `text` |
| `--output-mode` | `-m` | Output mode: `simple` or `reporting` | `simple` |
| `--target` | `-t` | Target OCP version for rules evaluation (e.g., 4.19) | highest in rules |
| `--rules` | `-r` | Path to rules.yaml file | `./rules.yaml` |
| `--version` | `-v` | Show version information | - |
| `--help` | `-h` | Show help | - |

### Examples

```bash
# Analyze for specific OCP version
rds-analyzer -i results.json -t 4.19

# Use custom rules file
rds-analyzer -i results.json -r /path/to/custom-rules.yaml

# Generate HTML report specifying OCP 4.20 release
rds-analyzer -i results.json -o html -t 4.20 > report-4.20.html

# Pipeline usage
compare-tool --output json | rds-analyzer -o html > analysis.html

# Generate reporting format specifying OCP 4.20 release
rds-analyzer -i results.json -m reporting -t 4.20
```

## Configuration

### Rules File Format

The rules are defined in `rules.yaml`. The file structure supports:

- **Global rules**: Apply to all CRs
- **Specific rules**: Match specific template/CR patterns
- **Count rules**: Check aggregate conditions (e.g., "only one CatalogSource allowed")
- **Label/Annotation rules**: Validate metadata with specificity-based matching
- **Version-specific impacts**: Different impacts for different OCP versions

Example rule:

```yaml
rules:
  - id: "R001-network-diagnostics"
    description: "Network diagnostics configuration validation"
    match:
      templateFileName: "DisableSnoNetworkDiag.yaml"
      crName: "operator.openshift.io/v1_Network_cluster"
    conditions:
      - type: "ExpectedFound"
        contains: "disableNetworkDiagnostics: false"
        impact:
          4.20: Impacting
        comment: "Network diagnostics should be disabled for RAN deployments"
```

### Condition Types

| Type | Description |
|------|-------------|
| `Any` | Match in any diff section |
| `FoundNotExpected` | Lines found but not in template |
| `ExpectedNotFound` | Lines expected but not found |
| `ExpectedFound` | Value differences (checks found value) |

### Matching

- `contains`: Simple substring match (supports multiline)
- `regex`: Regular expression pattern (takes precedence)

### Version-Specific Impacts

Impacts can vary by OCP version:

```yaml
impact:
  4.19: Impacting
  4.20: NotImpacting
  4.23: Impacting
```

The engine uses smart inheritance: if targeting 4.21, it inherits from 4.20 (the highest version <= target).

## Output Formats

### Text Output

Color-coded terminal output with:
- Validation summary
- Missing CRs with impact indicators
- Configuration differences with matched rules
- Count rule violations
- Impact statistics

### HTML Output

Interactive HTML report with:
- Collapsible sections
- Color-coded impact badges
- Rule tooltips with explanations
- Print-friendly styling

### Output Modes

| Mode | Description |
|------|-------------|
| `simple` | Default output showing all deviations and impacts |
| `reporting` | Structured output optimized for reporting workflows |

## Impact Levels

| Level | Symbol | Description |
|-------|--------|-------------|
| Impacting | Red | Deviation must be corrected |
| Not Impacting | Yellow | Deviation requires attention (RDS expansion) or support exception |
| Not a Deviation | Green | Configuration is compliant |
| Needs Review | Gray | No matching rule; requires manual review |

## Development

### Building

```bash
make build          # Build for current platform
make build-all      # Build for Linux, macOS, Windows
make test           # Run tests
make test-coverage  # Run tests with coverage report
make lint           # Run linter
make fmt            # Format code
make vet            # Run go vet
```

### Container Image Building

```bash
make image-build              # Build for current platform
make image-build-multiarch    # Build multi-arch image (amd64, arm64)
make image-push               # Push to registry
```

### Project Structure

```
rds-analyzer/
├── cmd/rds-analyzer/     # Entry point
├── internal/
│   ├── analyzer/         # Core analysis orchestration
│   ├── cli/              # Cobra CLI implementation
│   ├── parser/           # Diff parsing utilities
│   ├── report/           # Output generators (text, HTML)
│   ├── rules/            # Rule engine and types
│   └── types/            # Data structures
├── examples/             # Example rules and kube-compare output
├── docs/                 # Additional documentation
├── Dockerfile            # Container image definition
├── Makefile              # Build automation
└── README.md
```

## Contributing

See [AGENTS.md](AGENTS.md) for development guidelines and code conventions.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
