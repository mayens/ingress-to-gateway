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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/mayens/ingress-to-gateway/pkg/converter"
	"github.com/mayens/ingress-to-gateway/pkg/k8s"
)

var (
	batchOutputDir string
	batchNamespace string
	batchAll       bool
)

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch [flags]",
	Short: "Batch convert multiple Ingress resources",
	Long: `Batch converts multiple Ingress resources to HTTPRoutes with intelligent
grouping and organization.

The batch converter:
  • Groups Ingresses by namespace
  • Creates organized output directory structure
  • Applies consistent naming conventions
  • Generates summary report

Example usage:
  # Convert all Ingress in current namespace
  ingress-to-gateway batch -o ./output

  # Convert across all namespaces
  ingress-to-gateway batch --all-namespaces -o ./output

  # Batch convert with per-host splitting
  ingress-to-gateway batch --split-mode=per-host -o ./output`,
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)

	batchCmd.Flags().StringVarP(&batchOutputDir, "output-dir", "o", "./httproutes", "output directory for HTTPRoutes")
	batchCmd.Flags().BoolVarP(&batchAll, "all-namespaces", "A", false, "convert across all namespaces")
	batchCmd.Flags().StringVar(&splitMode, "split-mode", "single", "split mode: single, per-host, per-pattern")
	batchCmd.Flags().StringVar(&gatewayClass, "gateway-class", "nginx", "gateway class name")
}

func runBatch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Kubernetes client
	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Determine namespaces
	var namespaces []string
	if batchAll {
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

	// Create output directory
	if err := os.MkdirAll(batchOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create converter
	opts := converter.Options{
		SplitMode:    splitMode,
		GatewayClass: gatewayClass,
		OutputFormat: "yaml",
	}
	c := converter.NewConverter(opts)

	totalConverted := 0
	totalFailed := 0

	// Process each namespace
	for _, ns := range namespaces {
		fmt.Fprintf(os.Stderr, "Processing namespace: %s\n", ns)

		ingresses, err := client.ListIngresses(ctx, ns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list ingresses in %s: %v\n", ns, err)
			continue
		}

		if len(ingresses) == 0 {
			fmt.Fprintf(os.Stderr, "  No Ingress resources found\n")
			continue
		}

		// Create namespace subdirectory
		nsDir := filepath.Join(batchOutputDir, ns)
		if err := os.MkdirAll(nsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create directory %s: %v\n", nsDir, err)
			continue
		}

		// Convert each ingress
		for _, ingress := range ingresses {
			name := ingress.GetName()
			fmt.Fprintf(os.Stderr, "  Converting: %s\n", name)

			httpRoutes, err := c.Convert(ctx, []interface{}{ingress})
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Error: %v\n", err)
				totalFailed++
				continue
			}

			// Write HTTPRoutes
			for i, hr := range httpRoutes {
				filename := fmt.Sprintf("%s-httproute", name)
				if len(httpRoutes) > 1 {
					filename = fmt.Sprintf("%s-httproute-%d", name, i+1)
				}
				filename += ".yaml"

				outputPath := filepath.Join(nsDir, filename)
				f, err := os.Create(outputPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "    Error creating %s: %v\n", filename, err)
					totalFailed++
					continue
				}

				if err := c.WriteOutput([]interface{}{hr}, f); err != nil {
					f.Close()
					fmt.Fprintf(os.Stderr, "    Error writing %s: %v\n", filename, err)
					totalFailed++
					continue
				}
				f.Close()
				fmt.Fprintf(os.Stderr, "    Created: %s\n", filename)
				totalConverted++
			}
		}
	}

	// Summary
	fmt.Fprintf(os.Stderr, "\nBatch conversion complete:\n")
	fmt.Fprintf(os.Stderr, "  Successfully converted: %d\n", totalConverted)
	if totalFailed > 0 {
		fmt.Fprintf(os.Stderr, "  Failed: %d\n", totalFailed)
	}
	fmt.Fprintf(os.Stderr, "  Output directory: %s\n", batchOutputDir)

	return nil
}
