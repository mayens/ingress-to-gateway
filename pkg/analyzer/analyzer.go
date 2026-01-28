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
	"context"
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"github.com/mayens/ingress-to-gateway/pkg/k8s"
)

// Analyzer analyzes Ingress resources for migration readiness
type Analyzer struct {
	client *k8s.Client
}

// AnalysisResult contains the analysis results for an Ingress
type AnalysisResult struct {
	Name              string
	Namespace         string
	IngressClass      string
	HostCount         int
	Hostnames         []string
	PathCount         int
	TLSEnabled        bool
	Annotations       map[string]string
	DetectedFeatures  []string
	ComplexityScore   int
	MigrationReadiness string
	Issues            []string
	Recommendations   []string
}

// NewAnalyzer creates a new Analyzer
func NewAnalyzer(client *k8s.Client) *Analyzer {
	return &Analyzer{
		client: client,
	}
}

// AnalyzeIngresses analyzes all Ingress resources in specified namespaces
func (a *Analyzer) AnalyzeIngresses(ctx context.Context, namespaces []string) ([]*AnalysisResult, error) {
	var results []*AnalysisResult

	for _, ns := range namespaces {
		ingresses, err := a.client.ListIngresses(ctx, ns)
		if err != nil {
			return nil, fmt.Errorf("failed to list ingresses in %s: %w", ns, err)
		}

		for _, ing := range ingresses {
			result := a.analyzeIngress(ing)
			results = append(results, result)
		}
	}

	return results, nil
}

// analyzeIngress performs detailed analysis on a single Ingress
func (a *Analyzer) analyzeIngress(ing *networkingv1.Ingress) *AnalysisResult {
	result := &AnalysisResult{
		Name:         ing.Name,
		Namespace:    ing.Namespace,
		IngressClass: getIngressClass(ing),
		Annotations:  ing.Annotations,
		TLSEnabled:   len(ing.Spec.TLS) > 0,
	}

	// Count hosts and paths
	hostMap := make(map[string]bool)
	pathCount := 0

	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hostMap[rule.Host] = true
		}
		if rule.HTTP != nil {
			pathCount += len(rule.HTTP.Paths)
		}
	}

	result.HostCount = len(hostMap)
	for host := range hostMap {
		result.Hostnames = append(result.Hostnames, host)
	}
	result.PathCount = pathCount

	// Detect features and calculate complexity
	result.DetectedFeatures = a.detectFeatures(ing)
	result.ComplexityScore = a.calculateComplexity(ing, result.DetectedFeatures)
	result.MigrationReadiness = a.assessReadiness(result.ComplexityScore, result.DetectedFeatures)

	// Identify issues and recommendations
	result.Issues = a.identifyIssues(ing, result.DetectedFeatures)
	result.Recommendations = a.generateRecommendations(ing, result)

	return result
}

// detectFeatures detects NGINX Ingress features used
func (a *Analyzer) detectFeatures(ing *networkingv1.Ingress) []string {
	var features []string
	annotations := ing.Annotations

	// Check for 17+ annotations support
	annotationChecks := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target":         "URL_REWRITE",
		"nginx.ingress.kubernetes.io/app-root":               "APP_ROOT",
		"nginx.ingress.kubernetes.io/ssl-redirect":           "SSL_REDIRECT",
		"nginx.ingress.kubernetes.io/force-ssl-redirect":     "FORCE_SSL_REDIRECT",
		"nginx.ingress.kubernetes.io/permanent-redirect":     "PERMANENT_REDIRECT",
		"nginx.ingress.kubernetes.io/temporal-redirect":      "TEMPORAL_REDIRECT",
		"nginx.ingress.kubernetes.io/proxy-body-size":        "PROXY_BODY_SIZE",
		"nginx.ingress.kubernetes.io/proxy-read-timeout":     "PROXY_READ_TIMEOUT",
		"nginx.ingress.kubernetes.io/proxy-send-timeout":     "PROXY_SEND_TIMEOUT",
		"nginx.ingress.kubernetes.io/proxy-connect-timeout":  "PROXY_CONNECT_TIMEOUT",
		"nginx.ingress.kubernetes.io/backend-protocol":       "BACKEND_PROTOCOL",
		"nginx.ingress.kubernetes.io/cors-allow-origin":      "CORS",
		"nginx.ingress.kubernetes.io/enable-cors":            "CORS",
		"nginx.ingress.kubernetes.io/auth-type":              "AUTHENTICATION",
		"nginx.ingress.kubernetes.io/auth-secret":            "AUTHENTICATION",
		"nginx.ingress.kubernetes.io/canary":                 "CANARY",
		"nginx.ingress.kubernetes.io/canary-weight":          "CANARY_WEIGHT",
		"nginx.ingress.kubernetes.io/canary-by-header":       "CANARY_HEADER",
		"nginx.ingress.kubernetes.io/mirror-uri":             "MIRRORING",
		"nginx.ingress.kubernetes.io/mirror-target":          "MIRRORING",
		"nginx.ingress.kubernetes.io/configuration-snippet":  "CUSTOM_SNIPPET",
		"nginx.ingress.kubernetes.io/server-snippet":         "SERVER_SNIPPET",
	}

	for ann, feature := range annotationChecks {
		if _, exists := annotations[ann]; exists {
			if !contains(features, feature) {
				features = append(features, feature)
			}
		}
	}

	// Check TLS
	if len(ing.Spec.TLS) > 0 {
		features = append(features, "TLS_TERMINATION")
	}

	// Check for default backend
	if ing.Spec.DefaultBackend != nil {
		features = append(features, "DEFAULT_BACKEND")
	}

	return features
}

// calculateComplexity calculates migration complexity score
func (a *Analyzer) calculateComplexity(ing *networkingv1.Ingress, features []string) int {
	score := 0

	// Base complexity
	score += len(ing.Spec.Rules) * 2
	score += len(ing.Spec.TLS) * 3

	// Feature complexity
	complexityMap := map[string]int{
		"URL_REWRITE":       5,
		"CUSTOM_SNIPPET":    10,
		"SERVER_SNIPPET":    10,
		"CANARY":            7,
		"CANARY_WEIGHT":     7,
		"MIRRORING":         8,
		"AUTHENTICATION":    6,
		"CORS":              4,
		"BACKEND_PROTOCOL":  3,
		"PROXY_READ_TIMEOUT": 2,
		"SSL_REDIRECT":      2,
	}

	for _, feature := range features {
		if weight, exists := complexityMap[feature]; exists {
			score += weight
		}
	}

	return score
}

// assessReadiness determines migration readiness level
func (a *Analyzer) assessReadiness(score int, features []string) string {
	// Check for blockers
	blockers := []string{"CUSTOM_SNIPPET", "SERVER_SNIPPET"}
	for _, blocker := range blockers {
		if contains(features, blocker) {
			return "MANUAL_REVIEW_REQUIRED"
		}
	}

	// Score-based readiness
	if score <= 10 {
		return "READY"
	} else if score <= 25 {
		return "MOSTLY_READY"
	} else {
		return "COMPLEX"
	}
}

// identifyIssues identifies potential migration issues
func (a *Analyzer) identifyIssues(ing *networkingv1.Ingress, features []string) []string {
	var issues []string

	// Check for problematic annotations
	if contains(features, "CUSTOM_SNIPPET") || contains(features, "SERVER_SNIPPET") {
		issues = append(issues, "Custom NGINX snippets require manual review and cannot be directly migrated")
	}

	// Check for multiple IngressClasses
	if class := getIngressClass(ing); class != "" && !strings.Contains(class, "nginx") {
		issues = append(issues, fmt.Sprintf("Non-NGINX Ingress class detected: %s", class))
	}

	// Check for deprecated annotations
	deprecatedAnns := []string{
		"kubernetes.io/ingress.class",
	}
	for _, ann := range deprecatedAnns {
		if _, exists := ing.Annotations[ann]; exists {
			issues = append(issues, fmt.Sprintf("Using deprecated annotation: %s (use spec.ingressClassName instead)", ann))
		}
	}

	return issues
}

// generateRecommendations generates migration recommendations
func (a *Analyzer) generateRecommendations(ing *networkingv1.Ingress, result *AnalysisResult) []string {
	var recommendations []string

	// Splitting strategy
	if result.HostCount > 5 {
		recommendations = append(recommendations, "Consider using 'per-pattern' split mode for better organization")
	} else if result.HostCount > 1 {
		recommendations = append(recommendations, "Use 'single' split mode (default) for optimal Gateway API resource usage")
	}

	// Timeout recommendations
	if contains(result.DetectedFeatures, "PROXY_READ_TIMEOUT") {
		recommendations = append(recommendations, "Both timeouts.request and timeouts.backendRequest will be set for complete timeout control")
	}

	// Canary recommendations
	if contains(result.DetectedFeatures, "CANARY") {
		recommendations = append(recommendations, "Canary deployments will be converted to HTTPRoute backendRefs with traffic splitting")
	}

	// TLS recommendations
	if result.TLSEnabled {
		recommendations = append(recommendations, "Ensure Gateway has matching HTTPS listeners configured")
	}

	return recommendations
}

// getIngressClass extracts the Ingress class from spec or annotation
func getIngressClass(ing *networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return *ing.Spec.IngressClassName
	}
	if class, exists := ing.Annotations["kubernetes.io/ingress.class"]; exists {
		return class
	}
	return ""
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
