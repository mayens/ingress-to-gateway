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
	"github.com/mayens/ingress-to-gateway/pkg/validator"
)

var (
	validateFile string
	strict       bool
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate [file] [flags]",
	Short: "Validate HTTPRoute resources",
	Long: `Validate checks HTTPRoute resources for correctness and best practices.

The validator checks:
  • YAML/JSON syntax
  • Gateway API schema compliance
  • Reference validity (Gateway, Service)
  • Timeout constraints (backendRequest <= request)
  • Path match conflicts
  • Best practice recommendations

Example usage:
  # Validate a single file
  ingress-to-gateway validate httproute.yaml

  # Validate with strict mode (fail on warnings)
  ingress-to-gateway validate httproute.yaml --strict`,
	RunE: runValidate,
	Args: cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as errors")
}

func runValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	validateFile = args[0]

	// Create validator
	v := validator.NewValidator(strict)

	// Validate file
	results, err := v.ValidateFile(ctx, validateFile)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Print results
	hasErrors := false
	hasWarnings := false

	for _, result := range results {
		if len(result.Errors) > 0 {
			hasErrors = true
			fmt.Fprintf(os.Stderr, "❌ Errors in %s:\n", result.ResourceName)
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
		}
		if len(result.Warnings) > 0 {
			hasWarnings = true
			fmt.Fprintf(os.Stderr, "⚠️  Warnings in %s:\n", result.ResourceName)
			for _, w := range result.Warnings {
				fmt.Fprintf(os.Stderr, "  - %s\n", w)
			}
		}
	}

	if !hasErrors && !hasWarnings {
		fmt.Println("✅ Validation passed: No issues found")
		return nil
	}

	if hasErrors || (strict && hasWarnings) {
		return fmt.Errorf("validation failed")
	}

	fmt.Println("✅ Validation passed with warnings")
	return nil
}
