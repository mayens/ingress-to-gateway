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

package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mayens/ingress-to-gateway/pkg/analyzer"
	"sigs.k8s.io/yaml"
)

// Reporter generates reports for analysis results
type Reporter struct {
	format   string // table, json, yaml
	detailed bool
}

// NewReporter creates a new Reporter
func NewReporter(format string, detailed bool) *Reporter {
	return &Reporter{
		format:   format,
		detailed: detailed,
	}
}

// GenerateAuditReport generates an audit report
func (r *Reporter) GenerateAuditReport(results []*analyzer.AnalysisResult, w io.Writer) error {
	switch r.format {
	case "json":
		return r.generateJSONReport(results, w)
	case "yaml":
		return r.generateYAMLReport(results, w)
	default:
		return r.generateTableReport(results, w)
	}
}

// generateTableReport generates a table format report
func (r *Reporter) generateTableReport(results []*analyzer.AnalysisResult, w io.Writer) error {
	fmt.Fprintln(w, "╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(w, "║                      INGRESS MIGRATION AUDIT REPORT                           ║")
	fmt.Fprintln(w, "╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(w)

	// Summary
	fmt.Fprintf(w, "Total Ingress Resources: %d\n", len(results))
	fmt.Fprintln(w)

	// Readiness summary
	readinessCounts := make(map[string]int)
	for _, result := range results {
		readinessCounts[result.MigrationReadiness]++
	}

	fmt.Fprintln(w, "Migration Readiness Summary:")
	for readiness, count := range readinessCounts {
		icon := "✅"
		if readiness == "COMPLEX" {
			icon = "⚠️"
		} else if readiness == "MANUAL_REVIEW_REQUIRED" {
			icon = "❌"
		}
		fmt.Fprintf(w, "  %s %s: %d\n", icon, readiness, count)
	}
	fmt.Fprintln(w)

	// Detailed results
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(w, "INGRESS DETAILS")
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(w)

	for _, result := range results {
		r.printIngressDetail(result, w)
		fmt.Fprintln(w)
	}

	return nil
}

// printIngressDetail prints detailed information for a single Ingress
func (r *Reporter) printIngressDetail(result *analyzer.AnalysisResult, w io.Writer) {
	// Header
	fmt.Fprintf(w, "📋 Ingress: %s/%s\n", result.Namespace, result.Name)
	fmt.Fprintln(w, strings.Repeat("─", 80))

	// Basic info
	fmt.Fprintf(w, "  Ingress Class: %s\n", result.IngressClass)
	fmt.Fprintf(w, "  Hosts: %d | Paths: %d | TLS: %v\n", result.HostCount, result.PathCount, result.TLSEnabled)

	if len(result.Hostnames) > 0 {
		fmt.Fprintf(w, "  Hostnames: %s\n", strings.Join(result.Hostnames, ", "))
	}

	// Migration info
	icon := "✅"
	if result.MigrationReadiness == "COMPLEX" {
		icon = "⚠️"
	} else if result.MigrationReadiness == "MANUAL_REVIEW_REQUIRED" {
		icon = "❌"
	}
	fmt.Fprintf(w, "  Migration Readiness: %s %s (Complexity: %d)\n", icon, result.MigrationReadiness, result.ComplexityScore)

	// Features
	if len(result.DetectedFeatures) > 0 {
		fmt.Fprintf(w, "  Detected Features: %s\n", strings.Join(result.DetectedFeatures, ", "))
	}

	// Issues
	if len(result.Issues) > 0 {
		fmt.Fprintln(w, "  ⚠️  Issues:")
		for _, issue := range result.Issues {
			fmt.Fprintf(w, "    • %s\n", issue)
		}
	}

	// Recommendations (if detailed)
	if r.detailed && len(result.Recommendations) > 0 {
		fmt.Fprintln(w, "  💡 Recommendations:")
		for _, rec := range result.Recommendations {
			fmt.Fprintf(w, "    • %s\n", rec)
		}
	}
}

// generateJSONReport generates a JSON format report
func (r *Reporter) generateJSONReport(results []*analyzer.AnalysisResult, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// generateYAMLReport generates a YAML format report
func (r *Reporter) generateYAMLReport(results []*analyzer.AnalysisResult, w io.Writer) error {
	data, err := yaml.Marshal(results)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
