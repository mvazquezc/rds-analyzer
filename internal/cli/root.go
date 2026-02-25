// Package cli implements the command-line interface for rds-analyzer.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/openshift-kni/rds-analyzer/internal/analyzer"
	"github.com/openshift-kni/rds-analyzer/internal/types"
	"github.com/spf13/cobra"
)

// Version information set at build time.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Command-line flags.
var (
	outputFormat  string
	outputMode    string
	targetVersion string
	inputFile     string
	rulesFile     string
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "rds-analyzer",
	Short: "Analyze kube-compare JSON reports against a set of rules.",
	Long: `RDS Analyzer evaluates kube-compare JSON reports against a set of rules to determine the impact of configuration deviations from the reference configuration.

Examples:
  # Analyze from stdin with text output and using custom rules file
  cat results.json | rds-analyzer -r /path/to/custom-rules.yaml

  # Analyze from file with HTML output (using default rules file ./rules.yaml)
  rds-analyzer -i results.json -o html > report.html

  # Analyze using 4.19 OCP release for rules evaluation (using default rules file ./rules.yaml)
  rds-analyzer -i results.json -t 4.19

  # Use custom rules file and input file
  rds-analyzer -i results.json -r /path/to/rules.yaml

  # Generate reporting format
  rds-analyzer -i results.json -m reporting`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runAnalysis,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "text",
		"(Optional) Output format: text or html.")
	rootCmd.Flags().StringVarP(&outputMode, "output-mode", "m", "simple",
		"(Optional) Output mode: simple or reporting")
	rootCmd.Flags().StringVarP(&targetVersion, "target", "t", "",
		"(Optional) Target OCP version for rules evaluation (e.g., 4.19). If not specified, the highest available version in the rules will be used.")
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "",
		"Input file path (reads from stdin if not specified)")
	rootCmd.Flags().StringVarP(&rulesFile, "rules", "r", "./rules.yaml",
		"Path to rules.yaml file")

	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate)
}

// runAnalysis is the main execution function.
func runAnalysis(cmd *cobra.Command, args []string) error {
	if err := validateFlags(); err != nil {
		return err
	}

	validationReport, err := loadValidationReport()
	if err != nil {
		return err
	}

	a, err := analyzer.New(rulesFile, targetVersion)
	if err != nil {
		return err
	}

	return a.Analyze(os.Stdout, validationReport, outputFormat, outputMode)
}

// validateFlags validates command-line flag values.
func validateFlags() error {
	if outputFormat != "text" && outputFormat != "html" {
		return fmt.Errorf("invalid output format %q: must be 'text' or 'html'", outputFormat)
	}

	if outputMode != "simple" && outputMode != "reporting" {
		return fmt.Errorf("invalid output mode %q: must be 'simple' or 'reporting'", outputMode)
	}

	return nil
}

// loadValidationReport reads and parses the input validation report.
func loadValidationReport() (types.ValidationReport, error) {
	var report types.ValidationReport

	inputData, err := readInput()
	if err != nil {
		return report, fmt.Errorf("failed to read input: %w", err)
	}

	if len(inputData) == 0 {
		return report, fmt.Errorf("received empty input")
	}

	if err := json.Unmarshal(inputData, &report); err != nil {
		return report, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return report, nil
}

// readInput reads from file or stdin based on flags.
func readInput() ([]byte, error) {
	if inputFile != "" {
		return os.ReadFile(inputFile)
	}

	// Check if stdin has data.
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat stdin: %w", err)
	}

	// If stdin is a terminal (no pipe), show usage.
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, fmt.Errorf("no input provided; use -i flag or pipe JSON data")
	}

	return io.ReadAll(os.Stdin)
}
