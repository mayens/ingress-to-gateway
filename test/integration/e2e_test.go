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

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/mayens/ingress-to-gateway/pkg/converter"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/yaml"
)

// TestE2E_SimpleIngressConversion tests end-to-end conversion of a simple Ingress
func TestE2E_SimpleIngressConversion(t *testing.T) {
	// Load test fixture
	data, err := os.ReadFile("../fixtures/simple-ingress.yaml")
	if err != nil {
		t.Skipf("Skipping e2e test: fixture not found: %v", err)
		return
	}

	var ingress networkingv1.Ingress
	if err := yaml.Unmarshal(data, &ingress); err != nil {
		t.Fatalf("Failed to unmarshal ingress: %v", err)
	}

	// Convert
	opts := converter.Options{
		SplitMode:    "single",
		GatewayClass: "nginx",
		OutputFormat: "yaml",
	}
	c := converter.NewConverter(opts)

	ctx := context.Background()
	routes, err := c.Convert(ctx, []interface{}{&ingress})
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Verify results
	if len(routes) != 1 {
		t.Errorf("Expected 1 HTTPRoute, got %d", len(routes))
	}

	// Marshal to YAML to verify it's valid
	routeYAML, err := yaml.Marshal(routes[0])
	if err != nil {
		t.Fatalf("Failed to marshal HTTPRoute: %v", err)
	}

	// Basic checks
	routeStr := string(routeYAML)
	if !contains(routeStr, "app.example.com") {
		t.Error("HTTPRoute missing expected hostname")
	}
	if !contains(routeStr, "app-service") {
		t.Error("HTTPRoute missing expected service")
	}
	if !contains(routeStr, "request: 600s") {
		t.Error("HTTPRoute missing request timeout")
	}
	if !contains(routeStr, "backendRequest: 600s") {
		t.Error("HTTPRoute missing backendRequest timeout")
	}
}

// TestE2E_ComplexIngressConversion tests conversion with multiple annotations
func TestE2E_ComplexIngressConversion(t *testing.T) {
	data, err := os.ReadFile("../fixtures/complex-ingress.yaml")
	if err != nil {
		t.Skipf("Skipping e2e test: fixture not found: %v", err)
		return
	}

	var ingress networkingv1.Ingress
	if err := yaml.Unmarshal(data, &ingress); err != nil {
		t.Fatalf("Failed to unmarshal ingress: %v", err)
	}

	// Convert
	opts := converter.Options{
		SplitMode:    "single",
		GatewayClass: "nginx",
		OutputFormat: "yaml",
	}
	c := converter.NewConverter(opts)

	ctx := context.Background()
	routes, err := c.Convert(ctx, []interface{}{&ingress})
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("Expected 1 HTTPRoute, got %d", len(routes))
	}

	routeYAML, err := yaml.Marshal(routes[0])
	if err != nil {
		t.Fatalf("Failed to marshal HTTPRoute: %v", err)
	}

	routeStr := string(routeYAML)

	// Verify both hostnames present
	if !contains(routeStr, "app.example.com") {
		t.Error("HTTPRoute missing app.example.com")
	}
	if !contains(routeStr, "api.example.com") {
		t.Error("HTTPRoute missing api.example.com")
	}

	// Verify URL rewrite filter
	if !contains(routeStr, "URLRewrite") {
		t.Error("HTTPRoute missing URLRewrite filter")
	}

	// Verify timeouts
	if !contains(routeStr, "timeouts:") {
		t.Error("HTTPRoute missing timeouts")
	}
}

// TestE2E_PerHostSplit tests per-host splitting strategy
func TestE2E_PerHostSplit(t *testing.T) {
	data, err := os.ReadFile("../fixtures/complex-ingress.yaml")
	if err != nil {
		t.Skipf("Skipping e2e test: fixture not found: %v", err)
		return
	}

	var ingress networkingv1.Ingress
	if err := yaml.Unmarshal(data, &ingress); err != nil {
		t.Fatalf("Failed to unmarshal ingress: %v", err)
	}

	// Convert with per-host split
	opts := converter.Options{
		SplitMode:    "per-host",
		GatewayClass: "nginx",
		OutputFormat: "yaml",
	}
	c := converter.NewConverter(opts)

	ctx := context.Background()
	routes, err := c.Convert(ctx, []interface{}{&ingress})
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Should have 2 HTTPRoutes (one per host)
	if len(routes) != 2 {
		t.Errorf("Expected 2 HTTPRoutes for per-host mode, got %d", len(routes))
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
