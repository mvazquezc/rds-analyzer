// RDS Analyzer - Rule-based analyzer for OpenShift clusters comparisons.
//
// This tool evaluates kube-compare JSON reports against a set of rules to determine
// the impact of configuration deviations from the reference configuration.

package main

import (
	"fmt"
	"os"

	"github.com/openshift-kni/rds-analyzer/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
