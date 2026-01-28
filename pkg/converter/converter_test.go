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

package converter

import (
	"context"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestConvertSingle(t *testing.T) {
	ingress := createTestIngress()

	opts := Options{
		SplitMode:    "single",
		GatewayClass: "nginx",
		OutputFormat: "yaml",
	}
	c := NewConverter(opts)

	routes, err := c.convertSingle(ingress)
	if err != nil {
		t.Fatalf("convertSingle() error = %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("convertSingle() returned %v routes, want 1", len(routes))
	}

	route := routes[0].(*gatewayv1.HTTPRoute)

	// Verify hostnames
	if len(route.Spec.Hostnames) != 2 {
		t.Errorf("HTTPRoute has %v hostnames, want 2", len(route.Spec.Hostnames))
	}

	// Verify rules (should be deduplicated)
	if len(route.Spec.Rules) != 1 {
		t.Errorf("HTTPRoute has %v rules, want 1 (deduplicated)", len(route.Spec.Rules))
	}

	// Verify parent refs
	if len(route.Spec.ParentRefs) == 0 {
		t.Error("HTTPRoute has no parent refs")
	}
}

func TestConvertPerHost(t *testing.T) {
	ingress := createTestIngress()

	opts := Options{
		SplitMode:    "per-host",
		GatewayClass: "nginx",
		OutputFormat: "yaml",
	}
	c := NewConverter(opts)

	routes, err := c.convertPerHost(ingress)
	if err != nil {
		t.Fatalf("convertPerHost() error = %v", err)
	}

	// Should create one HTTPRoute per hostname
	if len(routes) != 2 {
		t.Errorf("convertPerHost() returned %v routes, want 2", len(routes))
	}

	// Each route should have exactly one hostname
	for i, r := range routes {
		route := r.(*gatewayv1.HTTPRoute)
		if len(route.Spec.Hostnames) != 1 {
			t.Errorf("Route %v has %v hostnames, want 1", i, len(route.Spec.Hostnames))
		}
	}
}

func TestExtractTimeouts(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		wantRequest bool
		wantBackend bool
	}{
		{
			name: "proxy-read-timeout",
			annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "600",
			},
			wantRequest: true,
			wantBackend: true,
		},
		{
			name: "proxy-send-timeout",
			annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-send-timeout": "600",
			},
			wantRequest: true,
			wantBackend: true,
		},
		{
			name: "both timeouts",
			annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "600",
				"nginx.ingress.kubernetes.io/proxy-send-timeout": "300",
			},
			wantRequest: true,
			wantBackend: true,
		},
		{
			name:        "no timeouts",
			annotations: map[string]string{},
			wantRequest: false,
			wantBackend: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingress := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.annotations,
				},
			}

			c := NewConverter(Options{})
			timeouts := c.extractTimeouts(ingress)

			if tt.wantRequest {
				if timeouts == nil || timeouts.Request == nil {
					t.Error("Expected request timeout, got nil")
				}
			}

			if tt.wantBackend {
				if timeouts == nil || timeouts.BackendRequest == nil {
					t.Error("Expected backendRequest timeout, got nil")
				}
			}

			// Verify both timeouts are set when any timeout is present
			if timeouts != nil {
				if timeouts.Request == nil {
					t.Error("request timeout not set")
				}
				if timeouts.BackendRequest == nil {
					t.Error("backendRequest timeout not set")
				}

				// Verify constraint: backendRequest <= request
				// In our implementation, they should be equal
				if *timeouts.BackendRequest != *timeouts.Request {
					t.Errorf("backendRequest (%v) != request (%v), want equal", *timeouts.BackendRequest, *timeouts.Request)
				}
			}
		})
	}
}

func TestExtractFilters(t *testing.T) {
	tests := []struct {
		name         string
		annotations  map[string]string
		wantFilters  int
		wantFilterType gatewayv1.HTTPRouteFilterType
	}{
		{
			name: "URL rewrite",
			annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			},
			wantFilters:    1,
			wantFilterType: gatewayv1.HTTPRouteFilterURLRewrite,
		},
		{
			name: "Permanent redirect",
			annotations: map[string]string{
				"nginx.ingress.kubernetes.io/permanent-redirect": "https://new-site.example.com",
			},
			wantFilters:    1,
			wantFilterType: gatewayv1.HTTPRouteFilterRequestRedirect,
		},
		{
			name:        "No filters",
			annotations: map[string]string{},
			wantFilters: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingress := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.annotations,
				},
			}

			c := NewConverter(Options{})
			filters, err := c.extractFilters(ingress)
			if err != nil {
				t.Fatalf("extractFilters() error = %v", err)
			}

			if len(filters) != tt.wantFilters {
				t.Errorf("extractFilters() returned %v filters, want %v", len(filters), tt.wantFilters)
			}

			if tt.wantFilters > 0 && filters[0].Type != tt.wantFilterType {
				t.Errorf("Filter type = %v, want %v", filters[0].Type, tt.wantFilterType)
			}
		})
	}
}

func TestDeriveGatewayName(t *testing.T) {
	tests := []struct {
		name        string
		ingress     *networkingv1.Ingress
		wantGateway string
	}{
		{
			name: "IngressClassName in spec",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					IngressClassName: stringPtr("nginx"),
				},
			},
			wantGateway: "gateway-nginx",
		},
		{
			name: "Annotation",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			wantGateway: "gateway-nginx",
		},
		{
			name:        "No class - default",
			ingress:     &networkingv1.Ingress{},
			wantGateway: "gateway-nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConverter(Options{})
			gateway := c.deriveGatewayName(tt.ingress)
			if gateway != tt.wantGateway {
				t.Errorf("deriveGatewayName() = %v, want %v", gateway, tt.wantGateway)
			}
		})
	}
}

func TestExtractHostPattern(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		wantPattern string
	}{
		{
			name:        "Simple domain",
			host:        "app.example.com",
			wantPattern: "example.com",
		},
		{
			name:        "Subdomain",
			host:        "api.dev.example.com",
			wantPattern: "example.com",
		},
		{
			name:        "Single segment",
			host:        "localhost",
			wantPattern: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConverter(Options{})
			pattern := c.extractHostPattern(tt.host)
			if pattern != tt.wantPattern {
				t.Errorf("extractHostPattern() = %v, want %v", pattern, tt.wantPattern)
			}
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{
			name:      "Valid name",
			input:     "example-com",
			wantValid: true,
		},
		{
			name:      "Uppercase to lowercase",
			input:     "EXAMPLE.COM",
			wantValid: true,
		},
		{
			name:      "Special characters",
			input:     "example_com!",
			wantValid: true,
		},
		{
			name:      "Very long name",
			input:     "very-long-name-that-exceeds-the-maximum-kubernetes-resource-name-length-limit-of-63-characters",
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeName(tt.input)

			// Check length
			if len(result) > 63 {
				t.Errorf("sanitizeName() result length = %v, want <= 63", len(result))
			}

			// Check format (lowercase alphanumeric and hyphens)
			for _, c := range result {
				if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
					t.Errorf("sanitizeName() result contains invalid character: %c", c)
				}
			}

			// Check doesn't start/end with hyphen
			if len(result) > 0 {
				if result[0] == '-' || result[len(result)-1] == '-' {
					t.Errorf("sanitizeName() result starts or ends with hyphen: %v", result)
				}
			}
		})
	}
}

func TestConvertHTTPRules(t *testing.T) {
	paths := []networkingv1.HTTPIngressPath{
		{
			Path:     "/",
			PathType: pathTypePtr(networkingv1.PathTypePrefix),
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: "app-service",
					Port: networkingv1.ServiceBackendPort{
						Number: 80,
					},
				},
			},
		},
		{
			Path:     "/api",
			PathType: pathTypePtr(networkingv1.PathTypePrefix),
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: "api-service",
					Port: networkingv1.ServiceBackendPort{
						Number: 8080,
					},
				},
			},
		},
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "600",
			},
		},
	}

	c := NewConverter(Options{})
	rules, err := c.convertHTTPRules(ingress, paths)
	if err != nil {
		t.Fatalf("convertHTTPRules() error = %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("convertHTTPRules() returned %v rules, want 2", len(rules))
	}

	// Verify each rule has timeouts
	for i, rule := range rules {
		if rule.Timeouts == nil {
			t.Errorf("Rule %v has no timeouts", i)
		}
		if len(rule.BackendRefs) == 0 {
			t.Errorf("Rule %v has no backend refs", i)
		}
		if len(rule.Matches) == 0 {
			t.Errorf("Rule %v has no matches", i)
		}
	}
}

func TestConvert(t *testing.T) {
	ingress := createTestIngress()

	tests := []struct {
		name      string
		splitMode string
		wantCount int
	}{
		{
			name:      "Single mode",
			splitMode: "single",
			wantCount: 1,
		},
		{
			name:      "Per-host mode",
			splitMode: "per-host",
			wantCount: 2,
		},
		{
			name:      "Per-pattern mode",
			splitMode: "per-pattern",
			wantCount: 1, // Both hosts have same pattern (example.com)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				SplitMode:    tt.splitMode,
				GatewayClass: "nginx",
				OutputFormat: "yaml",
			}
			c := NewConverter(opts)

			ctx := context.Background()
			routes, err := c.Convert(ctx, []interface{}{ingress})
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if len(routes) != tt.wantCount {
				t.Errorf("Convert() returned %v routes, want %v", len(routes), tt.wantCount)
			}
		})
	}
}

// Helper functions
func createTestIngress() *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "600",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPtr("nginx"),
			Rules: []networkingv1.IngressRule{
				{
					Host: "app.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pathTypePtr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "app-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "api.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pathTypePtr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "api-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func stringPtr(s string) *string {
	return &s
}

func pathTypePtr(pt networkingv1.PathType) *networkingv1.PathType {
	return &pt
}
