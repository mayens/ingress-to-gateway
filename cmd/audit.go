/*
Copyright 2026 The ingress-to-gateway Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/mayens/ingress-to-gateway/pkg/analyzer"
	"github.com/mayens/ingress-to-gateway/pkg/k8s"
	"github.com/mayens/ingress-to-gateway/pkg/reporter"
)

var (
	allNamespaces bool
	outputFormat  string
	detailed      bool
)

// auditCmd represents the audit command
var auditCmd = &cobra.Command{
	Use:   "audit [flags]",
	Short: "Audit Ingress resources for Gateway API migration readiness",
	Long: `Audit analyzes your Ingress resources and generates a comprehensive report
showing migration readiness, annotation usage, complexity scores, and potential issues.

The audit command helps you understand:
  • Which Ingress resources are in your cluster
  • What annotations are being used
  • Migration complexity scores
  • Potential migration blockers
  • Resource grouping recommendations

Example usage:
  # Audit all Ingress in current namespace
  ingress-to-gateway audit

  # Audit across all namespaces
  ingress-to-gateway audit --all-namespaces

  # Generate detailed report with JSON output
  ingress-to-gateway audit --detailed --output=json`,
	RunE: runAudit,
}

func init() {
	rootCmd.AddCommand(auditCmd)

	auditCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "audit across all namespaces")
	auditCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format: table, json, yaml")
	auditCmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "generate detailed report with recommendations")
}

func runAudit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Kubernetes client
	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Determine namespaces to audit
	var namespaces []string
	if allNamespaces {
		nsList, err := client.ListNamespaces(ctx)
		if err != nil {
			return fmt.Errorf("failed to list namespaces: %w", err)
		}
		namespaces = nsList
	} else {
		ns := namespace
		if ns == "" {
			ns, err = client.CurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to get current namespace: %w", err)
			}
		}
		namespaces = []string{ns}
	}

	// Create analyzer
	a := analyzer.NewAnalyzer(client)

	// Analyze ingresses
	fmt.Fprintf(os.Stderr, "Analyzing Ingress resources in %d namespace(s)...\n", len(namespaces))

	results, err := a.AnalyzeIngresses(ctx, namespaces)
	if err != nil {
		return fmt.Errorf("failed to analyze ingresses: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No Ingress resources found.")
		return nil
	}

	// Generate report
	r := reporter.NewReporter(outputFormat, detailed)
	if err := r.GenerateAuditReport(results, os.Stdout); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}
