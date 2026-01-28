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

package interactive

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mayens/ingress-to-gateway/pkg/analyzer"
	"github.com/mayens/ingress-to-gateway/pkg/converter"
	"github.com/mayens/ingress-to-gateway/pkg/k8s"
	"github.com/mayens/ingress-to-gateway/pkg/validator"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/yaml"
)

// Wizard provides an interactive migration experience
type Wizard struct {
	client   *k8s.Client
	reader   *bufio.Reader
	analyzer *analyzer.Analyzer
}

// NewWizard creates a new interactive wizard
func NewWizard(client *k8s.Client) *Wizard {
	return &Wizard{
		client:   client,
		reader:   bufio.NewReader(os.Stdin),
		analyzer: analyzer.NewAnalyzer(client),
	}
}

// Run starts the interactive wizard
func (w *Wizard) Run(ctx context.Context) error {
	w.printWelcome()

	// Step 1: Select namespace
	namespace, err := w.selectNamespace(ctx)
	if err != nil {
		return err
	}

	// Step 2: List and select Ingress
	ingress, err := w.selectIngress(ctx, namespace)
	if err != nil {
		return err
	}

	// Step 3: Analyze Ingress
	analysis, err := w.analyzer.AnalyzeIngresses(ctx, []string{namespace})
	if err != nil {
		return fmt.Errorf("failed to analyze: %w", err)
	}

	w.printAnalysis(analysis)

	// Step 4: Configure conversion options
	opts, err := w.configureOptions(ingress)
	if err != nil {
		return err
	}

	// Step 5: Preview conversion
	if err := w.previewConversion(ctx, ingress, opts); err != nil {
		return err
	}

	// Step 6: Confirm and save
	if err := w.confirmAndSave(ctx, ingress, opts); err != nil {
		return err
	}

	w.printSuccess()
	return nil
}

func (w *Wizard) printWelcome() {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║   Welcome to ingress-to-gateway Interactive Migration!    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This wizard will guide you through migrating your Ingress")
	fmt.Println("resources to Gateway API HTTPRoute step by step.")
	fmt.Println()
}

func (w *Wizard) selectNamespace(ctx context.Context) (string, error) {
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Step 1: Select Namespace")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	// Get current namespace
	current, err := w.client.CurrentNamespace()
	if err != nil {
		current = "default"
	}

	fmt.Printf("Current namespace: %s\n", current)
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  1. Use current namespace")
	fmt.Println("  2. List all namespaces")
	fmt.Println("  3. Enter namespace manually")
	fmt.Println()

	choice := w.prompt("Select option [1-3]")

	switch choice {
	case "1", "":
		return current, nil
	case "2":
		return w.selectFromNamespaceList(ctx)
	case "3":
		return w.prompt("Enter namespace name"), nil
	default:
		fmt.Println("Invalid choice, using current namespace")
		return current, nil
	}
}

func (w *Wizard) selectFromNamespaceList(ctx context.Context) (string, error) {
	namespaces, err := w.client.ListNamespaces(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list namespaces: %w", err)
	}

	fmt.Println()
	fmt.Println("Available namespaces:")
	for i, ns := range namespaces {
		fmt.Printf("  %d. %s\n", i+1, ns)
	}
	fmt.Println()

	choice := w.prompt(fmt.Sprintf("Select namespace [1-%d]", len(namespaces)))
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(namespaces) {
		fmt.Println("Invalid choice, using first namespace")
		return namespaces[0], nil
	}

	return namespaces[idx-1], nil
}

func (w *Wizard) selectIngress(ctx context.Context, namespace string) (*networkingv1.Ingress, error) {
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Step 2: Select Ingress Resource")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	ingresses, err := w.client.ListIngresses(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	if len(ingresses) == 0 {
		return nil, fmt.Errorf("no Ingress resources found in namespace %s", namespace)
	}

	fmt.Printf("Found %d Ingress resource(s) in namespace '%s':\n\n", len(ingresses), namespace)

	for i, ing := range ingresses {
		fmt.Printf("  %d. %s\n", i+1, ing.Name)
		fmt.Printf("     Class: %s\n", getIngressClass(ing))
		fmt.Printf("     Hosts: %d | Rules: %d | TLS: %v\n",
			len(getHosts(ing)),
			len(ing.Spec.Rules),
			len(ing.Spec.TLS) > 0)
		fmt.Println()
	}

	choice := w.prompt(fmt.Sprintf("Select Ingress [1-%d]", len(ingresses)))
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(ingresses) {
		fmt.Println("Invalid choice, using first Ingress")
		return ingresses[0], nil
	}

	return ingresses[idx-1], nil
}

func (w *Wizard) printAnalysis(results []*analyzer.AnalysisResult) {
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Step 3: Migration Analysis")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	if len(results) == 0 {
		return
	}

	result := results[0]

	// Basic info
	fmt.Printf("Ingress: %s/%s\n", result.Namespace, result.Name)
	fmt.Printf("Ingress Class: %s\n", result.IngressClass)
	fmt.Printf("Hosts: %d | Paths: %d | TLS: %v\n\n",
		result.HostCount, result.PathCount, result.TLSEnabled)

	// Features
	if len(result.DetectedFeatures) > 0 {
		fmt.Println("✓ Detected Features:")
		for _, feature := range result.DetectedFeatures {
			fmt.Printf("  • %s\n", feature)
		}
		fmt.Println()
	}

	// Readiness
	icon := "✅"
	if result.MigrationReadiness == "COMPLEX" {
		icon = "⚠️"
	} else if result.MigrationReadiness == "MANUAL_REVIEW_REQUIRED" {
		icon = "❌"
	}
	fmt.Printf("%s Migration Readiness: %s (Complexity: %d)\n\n",
		icon, result.MigrationReadiness, result.ComplexityScore)

	// Issues
	if len(result.Issues) > 0 {
		fmt.Println("⚠️  Issues:")
		for _, issue := range result.Issues {
			fmt.Printf("  • %s\n", issue)
		}
		fmt.Println()
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Println("💡 Recommendations:")
		for _, rec := range result.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
		fmt.Println()
	}

	w.prompt("Press Enter to continue...")
}

func (w *Wizard) configureOptions(ingress *networkingv1.Ingress) (*converter.Options, error) {
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Step 4: Configure Conversion Options")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	opts := &converter.Options{
		OutputFormat: "yaml",
	}

	// Split mode
	fmt.Println("Select split mode:")
	fmt.Println("  1. Single   - One HTTPRoute for all hostnames (recommended)")
	fmt.Println("  2. Per-host - Separate HTTPRoute per hostname")
	fmt.Println("  3. Per-pattern - Group by hostname patterns")
	fmt.Println()

	splitChoice := w.prompt("Split mode [1-3]")
	switch splitChoice {
	case "2":
		opts.SplitMode = "per-host"
	case "3":
		opts.SplitMode = "per-pattern"
	default:
		opts.SplitMode = "single"
	}
	fmt.Printf("✓ Split mode: %s\n\n", opts.SplitMode)

	// Gateway name
	defaultGateway := "gateway-nginx"
	if ingress.Spec.IngressClassName != nil {
		defaultGateway = fmt.Sprintf("gateway-%s", *ingress.Spec.IngressClassName)
	}

	fmt.Printf("Gateway name (default: %s):\n", defaultGateway)
	gatewayName := w.prompt("Gateway name (Enter for default)")
	if gatewayName == "" {
		opts.GatewayName = defaultGateway
	} else {
		opts.GatewayName = gatewayName
	}
	fmt.Printf("✓ Gateway: %s\n\n", opts.GatewayName)

	// Gateway class
	defaultClass := "nginx"
	fmt.Printf("Gateway class (default: %s):\n", defaultClass)
	gatewayClass := w.prompt("Gateway class (Enter for default)")
	if gatewayClass == "" {
		opts.GatewayClass = defaultClass
	} else {
		opts.GatewayClass = gatewayClass
	}
	fmt.Printf("✓ Gateway class: %s\n\n", opts.GatewayClass)

	return opts, nil
}

func (w *Wizard) previewConversion(ctx context.Context, ingress *networkingv1.Ingress, opts *converter.Options) error {
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Step 5: Preview Conversion")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	// Convert
	c := converter.NewConverter(*opts)
	routes, err := c.Convert(ctx, []interface{}{ingress})
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	fmt.Printf("✓ Generated %d HTTPRoute(s)\n\n", len(routes))

	// Preview first route
	if len(routes) > 0 {
		routeYAML, err := yaml.Marshal(routes[0])
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}

		fmt.Println("Preview (first HTTPRoute):")
		fmt.Println("───────────────────────────────────────────────────────────")
		fmt.Println(string(routeYAML))
		fmt.Println("───────────────────────────────────────────────────────────")
		fmt.Println()

		if len(routes) > 1 {
			fmt.Printf("Note: %d additional HTTPRoute(s) will be generated.\n\n", len(routes)-1)
		}
	}

	w.prompt("Press Enter to continue...")
	return nil
}

func (w *Wizard) confirmAndSave(ctx context.Context, ingress *networkingv1.Ingress, opts *converter.Options) error {
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Step 6: Confirm and Save")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	// Convert
	c := converter.NewConverter(*opts)
	routes, err := c.Convert(ctx, []interface{}{ingress})
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Validate
	fmt.Println("Validating HTTPRoute(s)...")
	v := validator.NewValidator(false)
	hasErrors := false

	for _, route := range routes {
		routeYAML, _ := yaml.Marshal(route)
		results, err := v.ValidateFile(ctx, string(routeYAML))
		if err != nil {
			fmt.Printf("⚠️  Validation error: %v\n", err)
			continue
		}

		for _, result := range results {
			if len(result.Errors) > 0 {
				hasErrors = true
				fmt.Printf("❌ Errors in %s:\n", result.ResourceName)
				for _, e := range result.Errors {
					fmt.Printf("  • %s\n", e)
				}
			}
		}
	}

	if hasErrors {
		fmt.Println()
		fmt.Println("⚠️  Validation found errors. Review and fix before applying.")
	} else {
		fmt.Println("✓ Validation passed\n")
	}

	// Confirm
	fmt.Println("What would you like to do?")
	fmt.Println("  1. Save to file")
	fmt.Println("  2. Print to stdout")
	fmt.Println("  3. Cancel")
	fmt.Println()

	choice := w.prompt("Select option [1-3]")

	switch choice {
	case "1":
		return w.saveToFile(routes, ingress.Name)
	case "2":
		return w.printToStdout(routes)
	case "3":
		fmt.Println("Migration cancelled.")
		return nil
	default:
		fmt.Println("Invalid choice, printing to stdout...")
		return w.printToStdout(routes)
	}
}

func (w *Wizard) saveToFile(routes []interface{}, baseName string) error {
	defaultFilename := fmt.Sprintf("%s-httproute.yaml", baseName)
	fmt.Printf("Output filename (default: %s):\n", defaultFilename)
	filename := w.prompt("Filename (Enter for default)")
	if filename == "" {
		filename = defaultFilename
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	for i, route := range routes {
		if i > 0 {
			fmt.Fprintln(f, "---")
		}
		routeYAML, err := yaml.Marshal(route)
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}
		f.Write(routeYAML)
	}

	fmt.Printf("\n✓ Saved to: %s\n", filename)
	return nil
}

func (w *Wizard) printToStdout(routes []interface{}) error {
	fmt.Println()
	for i, route := range routes {
		if i > 0 {
			fmt.Println("---")
		}
		routeYAML, err := yaml.Marshal(route)
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}
		fmt.Print(string(routeYAML))
	}
	return nil
}

func (w *Wizard) printSuccess() {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Migration Completed Successfully!            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review the generated HTTPRoute(s)")
	fmt.Println("  2. Ensure Gateway resource exists in your cluster")
	fmt.Println("  3. Apply: kubectl apply -f <httproute-file>")
	fmt.Println("  4. Test traffic routing")
	fmt.Println("  5. Monitor for any issues")
	fmt.Println()
	fmt.Println("Thank you for using ingress-to-gateway! 🚀")
	fmt.Println()
}

func (w *Wizard) prompt(message string) string {
	fmt.Printf("%s: ", message)
	input, _ := w.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// Helper functions
func getIngressClass(ing *networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return *ing.Spec.IngressClassName
	}
	if class, ok := ing.Annotations["kubernetes.io/ingress.class"]; ok {
		return class
	}
	return ""
}

func getHosts(ing *networkingv1.Ingress) []string {
	hosts := make(map[string]bool)
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hosts[rule.Host] = true
		}
	}
	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result
}
