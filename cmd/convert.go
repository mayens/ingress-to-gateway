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
	"github.com/mayens/ingress-to-gateway/pkg/converter"
	"github.com/mayens/ingress-to-gateway/pkg/k8s"
)

var (
	inputFile     string
	outputFile    string
	splitMode     string
	gatewayName   string
	gatewayClass  string
	convertOutput string
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert [ingress-name] [flags]",
	Short: "Convert Ingress to Gateway API HTTPRoute",
	Long: `Convert translates an Ingress resource to Gateway API HTTPRoute with full
support for 17+ NGINX Ingress annotations.

The converter intelligently handles:
  • Path-based routing with PathPrefix and Exact matches
  • TLS termination and certificate references
  • Timeouts (both request and backendRequest)
  • URL rewrites and redirects
  • Canary deployments with traffic splitting
  • CORS policies
  • Backend protocol (HTTP/HTTPS)
  • Custom headers and authentication

Split Modes:
  • single:      One HTTPRoute for all hostnames (optimized, default)
  • per-host:    Separate HTTPRoute per hostname (maximum flexibility)
  • per-pattern: Grouped by hostname patterns (intelligent)

Example usage:
  # Convert from cluster
  ingress-to-gateway convert my-ingress -n default

  # Convert from file with per-host splitting
  ingress-to-gateway convert -f ingress.yaml --split-mode=per-host

  # Convert and save to file
  ingress-to-gateway convert my-ingress -o httproute.yaml

  # Convert with custom gateway reference
  ingress-to-gateway convert my-ingress --gateway=my-gateway`,
	RunE: runConvert,
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&inputFile, "file", "f", "", "input file containing Ingress resource")
	convertCmd.Flags().StringVarP(&outputFile, "output-file", "o", "", "output file for HTTPRoute (default: stdout)")
	convertCmd.Flags().StringVar(&splitMode, "split-mode", "single", "split mode: single, per-host, per-pattern")
	convertCmd.Flags().StringVar(&gatewayName, "gateway", "", "gateway name to reference (default: derive from ingress class)")
	convertCmd.Flags().StringVar(&gatewayClass, "gateway-class", "nginx", "gateway class name")
	convertCmd.Flags().StringVar(&convertOutput, "format", "yaml", "output format: yaml or json")
}

func runConvert(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate split mode
	validModes := map[string]bool{"single": true, "per-host": true, "per-pattern": true}
	if !validModes[splitMode] {
		return fmt.Errorf("invalid split mode: %s (valid: single, per-host, per-pattern)", splitMode)
	}

	// Create converter
	opts := converter.Options{
		SplitMode:    splitMode,
		GatewayName:  gatewayName,
		GatewayClass: gatewayClass,
		OutputFormat: convertOutput,
	}
	c := converter.NewConverter(opts)

	var ingresses []interface{}
	var err error

	if inputFile != "" {
		// Read from file
		ingresses, err = c.LoadFromFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to load ingress from file: %w", err)
		}
	} else {
		// Read from cluster
		if len(args) == 0 {
			return fmt.Errorf("ingress name required when not using --file")
		}
		ingressName := args[0]

		client, err := k8s.NewClient(kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create kubernetes client: %w", err)
		}

		ns := namespace
		if ns == "" {
			ns, err = client.CurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to get current namespace: %w", err)
			}
		}

		ingress, err := client.GetIngress(ctx, ns, ingressName)
		if err != nil {
			return fmt.Errorf("failed to get ingress: %w", err)
		}
		ingresses = []interface{}{ingress}
	}

	// Convert to HTTPRoute
	httpRoutes, err := c.Convert(ctx, ingresses)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Output results
	output := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		output = f
	}

	if err := c.WriteOutput(httpRoutes, output); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if outputFile != "" {
		fmt.Fprintf(os.Stderr, "HTTPRoute(s) written to %s\n", outputFile)
	}

	return nil
}
