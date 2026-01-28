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

package analyzer

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetectFeatures(t *testing.T) {
	tests := []struct {
		name        string
		ingress     *networkingv1.Ingress
		wantFeatures []string
	}{
		{
			name: "URL rewrite annotation",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
					},
				},
			},
			wantFeatures: []string{"URL_REWRITE"},
		},
		{
			name: "TLS termination",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{
							Hosts:      []string{"example.com"},
							SecretName: "tls-secret",
						},
					},
				},
			},
			wantFeatures: []string{"TLS_TERMINATION"},
		},
		{
			name: "Multiple features",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"nginx.ingress.kubernetes.io/rewrite-target":       "/$2",
						"nginx.ingress.kubernetes.io/proxy-read-timeout":   "600",
						"nginx.ingress.kubernetes.io/enable-cors":          "true",
						"nginx.ingress.kubernetes.io/canary":               "true",
						"nginx.ingress.kubernetes.io/canary-weight":        "20",
					},
				},
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{
							Hosts:      []string{"example.com"},
							SecretName: "tls-secret",
						},
					},
				},
			},
			wantFeatures: []string{"URL_REWRITE", "PROXY_READ_TIMEOUT", "CORS", "CANARY", "CANARY_WEIGHT", "TLS_TERMINATION"},
		},
		{
			name: "Custom snippets",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"nginx.ingress.kubernetes.io/configuration-snippet": "proxy_set_header X-Custom-Header value;",
					},
				},
			},
			wantFeatures: []string{"CUSTOM_SNIPPET"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAnalyzer(nil)
			features := a.detectFeatures(tt.ingress)

			// Check all expected features are present
			for _, want := range tt.wantFeatures {
				found := false
				for _, got := range features {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("detectFeatures() missing feature %v, got %v", want, features)
				}
			}
		})
	}
}

func TestCalculateComplexity(t *testing.T) {
	tests := []struct {
		name       string
		ingress    *networkingv1.Ingress
		features   []string
		wantScore  int
		wantMin    int
		wantMax    int
	}{
		{
			name: "Simple ingress",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "example.com"},
					},
				},
			},
			features: []string{},
			wantMin:  0,
			wantMax:  5,
		},
		{
			name: "Ingress with TLS",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "example.com"},
					},
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"example.com"}},
					},
				},
			},
			features: []string{"TLS_TERMINATION"},
			wantMin:  3,
			wantMax:  10,
		},
		{
			name: "Complex ingress with snippets",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "example.com"},
						{Host: "api.example.com"},
						{Host: "www.example.com"},
					},
				},
			},
			features: []string{"CUSTOM_SNIPPET", "URL_REWRITE", "CANARY"},
			wantMin:  15,
			wantMax:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAnalyzer(nil)
			score := a.calculateComplexity(tt.ingress, tt.features)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("calculateComplexity() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestAssessReadiness(t *testing.T) {
	tests := []struct {
		name         string
		score        int
		features     []string
		wantReadiness string
	}{
		{
			name:         "Ready - low complexity",
			score:        5,
			features:     []string{"TLS_TERMINATION"},
			wantReadiness: "READY",
		},
		{
			name:         "Mostly ready - medium complexity",
			score:        15,
			features:     []string{"URL_REWRITE", "CORS"},
			wantReadiness: "MOSTLY_READY",
		},
		{
			name:         "Complex - high complexity",
			score:        30,
			features:     []string{"CANARY", "URL_REWRITE", "MIRRORING"},
			wantReadiness: "COMPLEX",
		},
		{
			name:         "Manual review - has snippets",
			score:        10,
			features:     []string{"CUSTOM_SNIPPET"},
			wantReadiness: "MANUAL_REVIEW_REQUIRED",
		},
		{
			name:         "Manual review - has server snippets",
			score:        8,
			features:     []string{"SERVER_SNIPPET"},
			wantReadiness: "MANUAL_REVIEW_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAnalyzer(nil)
			readiness := a.assessReadiness(tt.score, tt.features)

			if readiness != tt.wantReadiness {
				t.Errorf("assessReadiness() = %v, want %v", readiness, tt.wantReadiness)
			}
		})
	}
}

func TestIdentifyIssues(t *testing.T) {
	tests := []struct {
		name       string
		ingress    *networkingv1.Ingress
		features   []string
		wantIssues int
	}{
		{
			name: "No issues",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ingress",
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: stringPtr("nginx"),
				},
			},
			features:   []string{"TLS_TERMINATION"},
			wantIssues: 0,
		},
		{
			name: "Custom snippets issue",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ingress",
				},
			},
			features:   []string{"CUSTOM_SNIPPET"},
			wantIssues: 1,
		},
		{
			name: "Deprecated annotation",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ingress",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			features:   []string{},
			wantIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAnalyzer(nil)
			issues := a.identifyIssues(tt.ingress, tt.features)

			if len(issues) != tt.wantIssues {
				t.Errorf("identifyIssues() returned %v issues, want %v. Issues: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name                string
		ingress            *networkingv1.Ingress
		result             *AnalysisResult
		wantRecommendations int
	}{
		{
			name: "Single host - single mode recommended",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "example.com"},
					},
				},
			},
			result: &AnalysisResult{
				HostCount: 1,
			},
			wantRecommendations: 1, // At least one recommendation
		},
		{
			name: "Multiple hosts - should recommend split strategy",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "app1.example.com"},
						{Host: "app2.example.com"},
						{Host: "app3.example.com"},
					},
				},
			},
			result: &AnalysisResult{
				HostCount: 3,
			},
			wantRecommendations: 1,
		},
		{
			name: "Many hosts - per-pattern recommended",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: make([]networkingv1.IngressRule, 10),
				},
			},
			result: &AnalysisResult{
				HostCount: 10,
			},
			wantRecommendations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAnalyzer(nil)
			recommendations := a.generateRecommendations(tt.ingress, tt.result)

			if len(recommendations) < tt.wantRecommendations {
				t.Errorf("generateRecommendations() returned %v recommendations, want at least %v", len(recommendations), tt.wantRecommendations)
			}
		})
	}
}

func TestGetIngressClass(t *testing.T) {
	tests := []struct {
		name      string
		ingress   *networkingv1.Ingress
		wantClass string
	}{
		{
			name: "IngressClassName in spec",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					IngressClassName: stringPtr("nginx"),
				},
			},
			wantClass: "nginx",
		},
		{
			name: "Deprecated annotation",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			wantClass: "nginx",
		},
		{
			name:      "No class specified",
			ingress:   &networkingv1.Ingress{},
			wantClass: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class := getIngressClass(tt.ingress)
			if class != tt.wantClass {
				t.Errorf("getIngressClass() = %v, want %v", class, tt.wantClass)
			}
		})
	}
}

func TestAnalyzeIngress(t *testing.T) {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target":     "/$2",
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
									Path: "/",
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
									Path: "/",
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
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"app.example.com", "api.example.com"},
					SecretName: "tls-secret",
				},
			},
		},
	}

	a := NewAnalyzer(nil)
	result := a.analyzeIngress(ingress)

	// Verify basic fields
	if result.Name != "test-ingress" {
		t.Errorf("Name = %v, want test-ingress", result.Name)
	}

	if result.Namespace != "default" {
		t.Errorf("Namespace = %v, want default", result.Namespace)
	}

	if result.IngressClass != "nginx" {
		t.Errorf("IngressClass = %v, want nginx", result.IngressClass)
	}

	if result.HostCount != 2 {
		t.Errorf("HostCount = %v, want 2", result.HostCount)
	}

	if result.PathCount != 2 {
		t.Errorf("PathCount = %v, want 2", result.PathCount)
	}

	if !result.TLSEnabled {
		t.Error("TLSEnabled = false, want true")
	}

	// Verify features detected
	expectedFeatures := []string{"URL_REWRITE", "PROXY_READ_TIMEOUT", "TLS_TERMINATION"}
	for _, expected := range expectedFeatures {
		found := false
		for _, feature := range result.DetectedFeatures {
			if feature == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected feature: %v", expected)
		}
	}

	// Verify complexity score is calculated
	if result.ComplexityScore == 0 {
		t.Error("ComplexityScore = 0, expected non-zero value")
	}

	// Verify readiness is assessed
	if result.MigrationReadiness == "" {
		t.Error("MigrationReadiness is empty")
	}

	// Verify recommendations are generated
	if len(result.Recommendations) == 0 {
		t.Error("No recommendations generated")
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
