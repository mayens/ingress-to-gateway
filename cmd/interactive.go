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

	"github.com/mayens/ingress-to-gateway/pkg/interactive"
	"github.com/mayens/ingress-to-gateway/pkg/k8s"
	"github.com/spf13/cobra"
)

// interactiveCmd represents the interactive command
var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Interactive migration wizard",
	Long: `Interactive mode provides a step-by-step guided experience for migrating
your Ingress resources to Gateway API HTTPRoute.

The wizard will:
  1. Help you select a namespace and Ingress resource
  2. Analyze the Ingress for migration readiness
  3. Guide you through configuration options
  4. Preview the generated HTTPRoute
  5. Validate and save the output

This mode is ideal for:
  • First-time users learning the migration process
  • Complex migrations requiring careful review
  • Exploratory analysis of migration options
  • Users who prefer guided workflows

Example usage:
  # Start the interactive wizard
  ingress-to-gateway interactive

  # Use with specific kubeconfig
  ingress-to-gateway interactive --kubeconfig ~/.kube/prod-config`,
	RunE: runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

func runInteractive(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Kubernetes client
	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		return err
	}

	// Create and run wizard
	wizard := interactive.NewWizard(client)
	return wizard.Run(ctx)
}
