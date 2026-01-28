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
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Options contains converter configuration
type Options struct {
	SplitMode    string // single, per-host, per-pattern
	GatewayName  string
	GatewayClass string
	OutputFormat string // yaml, json
}

// Converter handles Ingress to HTTPRoute conversion
type Converter struct {
	opts Options
}

// NewConverter creates a new Converter
func NewConverter(opts Options) *Converter {
	return &Converter{
		opts: opts,
	}
}

// LoadFromFile loads Ingress resources from a file
func (c *Converter) LoadFromFile(path string) ([]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var ingress networkingv1.Ingress
	if err := yaml.Unmarshal(data, &ingress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ingress: %w", err)
	}

	return []interface{}{&ingress}, nil
}

// Convert converts Ingress resources to HTTPRoutes
func (c *Converter) Convert(ctx context.Context, ingresses []interface{}) ([]interface{}, error) {
	var httpRoutes []interface{}

	for _, ing := range ingresses {
		ingress, ok := ing.(*networkingv1.Ingress)
		if !ok {
			return nil, fmt.Errorf("invalid ingress type")
		}

		routes, err := c.convertIngress(ingress)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ingress %s: %w", ingress.Name, err)
		}

		httpRoutes = append(httpRoutes, routes...)
	}

	return httpRoutes, nil
}

// convertIngress converts a single Ingress to HTTPRoute(s)
func (c *Converter) convertIngress(ing *networkingv1.Ingress) ([]interface{}, error) {
	switch c.opts.SplitMode {
	case "single":
		return c.convertSingle(ing)
	case "per-host":
		return c.convertPerHost(ing)
	case "per-pattern":
		return c.convertPerPattern(ing)
	default:
		return nil, fmt.Errorf("invalid split mode: %s", c.opts.SplitMode)
	}
}

// convertSingle creates one HTTPRoute for all hosts
func (c *Converter) convertSingle(ing *networkingv1.Ingress) ([]interface{}, error) {
	httpRoute := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1",
			Kind:       "HTTPRoute",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-httproute", ing.Name),
			Namespace: ing.Namespace,
			Labels:    ing.Labels,
		},
	}

	// Collect all hostnames
	var hostnames []gatewayv1.Hostname
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hostnames = append(hostnames, gatewayv1.Hostname(rule.Host))
		}
	}
	httpRoute.Spec.Hostnames = hostnames

	// Set parent refs (Gateway)
	gatewayName := c.opts.GatewayName
	if gatewayName == "" {
		gatewayName = c.deriveGatewayName(ing)
	}
	httpRoute.Spec.ParentRefs = []gatewayv1.ParentReference{
		{
			Name: gatewayv1.ObjectName(gatewayName),
		},
	}

	// Convert rules (deduplicated - process only first rule's paths)
	if len(ing.Spec.Rules) > 0 && ing.Spec.Rules[0].HTTP != nil {
		rules, err := c.convertHTTPRules(ing, ing.Spec.Rules[0].HTTP.Paths)
		if err != nil {
			return nil, err
		}
		httpRoute.Spec.Rules = rules
	}

	// Handle default backend
	if ing.Spec.DefaultBackend != nil {
		defaultRule := c.createDefaultBackendRule(ing, ing.Spec.DefaultBackend)
		httpRoute.Spec.Rules = append(httpRoute.Spec.Rules, defaultRule)
	}

	return []interface{}{httpRoute}, nil
}

// convertPerHost creates separate HTTPRoute per hostname
func (c *Converter) convertPerHost(ing *networkingv1.Ingress) ([]interface{}, error) {
	var httpRoutes []interface{}

	for i, rule := range ing.Spec.Rules {
		if rule.Host == "" {
			continue
		}

		httpRoute := &gatewayv1.HTTPRoute{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "gateway.networking.k8s.io/v1",
				Kind:       "HTTPRoute",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-httproute-%d", ing.Name, i+1),
				Namespace: ing.Namespace,
				Labels:    ing.Labels,
			},
			Spec: gatewayv1.HTTPRouteSpec{
				Hostnames: []gatewayv1.Hostname{gatewayv1.Hostname(rule.Host)},
			},
		}

		// Set parent refs
		gatewayName := c.opts.GatewayName
		if gatewayName == "" {
			gatewayName = c.deriveGatewayName(ing)
		}
		httpRoute.Spec.ParentRefs = []gatewayv1.ParentReference{
			{
				Name: gatewayv1.ObjectName(gatewayName),
			},
		}

		// Convert rules
		if rule.HTTP != nil {
			rules, err := c.convertHTTPRules(ing, rule.HTTP.Paths)
			if err != nil {
				return nil, err
			}
			httpRoute.Spec.Rules = rules
		}

		httpRoutes = append(httpRoutes, httpRoute)
	}

	return httpRoutes, nil
}

// convertPerPattern groups hosts by pattern
func (c *Converter) convertPerPattern(ing *networkingv1.Ingress) ([]interface{}, error) {
	// Group hosts by pattern (e.g., *.example.com, *.dev.example.com)
	groups := make(map[string][]string)

	for _, rule := range ing.Spec.Rules {
		if rule.Host == "" {
			continue
		}

		pattern := c.extractHostPattern(rule.Host)
		groups[pattern] = append(groups[pattern], rule.Host)
	}

	var httpRoutes []interface{}
	i := 0

	for pattern, hosts := range groups {
		httpRoute := &gatewayv1.HTTPRoute{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "gateway.networking.k8s.io/v1",
				Kind:       "HTTPRoute",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-httproute-%s", ing.Name, sanitizeName(pattern)),
				Namespace: ing.Namespace,
				Labels:    ing.Labels,
			},
		}

		// Add hostnames
		for _, host := range hosts {
			httpRoute.Spec.Hostnames = append(httpRoute.Spec.Hostnames, gatewayv1.Hostname(host))
		}

		// Set parent refs
		gatewayName := c.opts.GatewayName
		if gatewayName == "" {
			gatewayName = c.deriveGatewayName(ing)
		}
		httpRoute.Spec.ParentRefs = []gatewayv1.ParentReference{
			{
				Name: gatewayv1.ObjectName(gatewayName),
			},
		}

		// Convert rules (use first rule's paths)
		if len(ing.Spec.Rules) > 0 && ing.Spec.Rules[0].HTTP != nil {
			rules, err := c.convertHTTPRules(ing, ing.Spec.Rules[0].HTTP.Paths)
			if err != nil {
				return nil, err
			}
			httpRoute.Spec.Rules = rules
		}

		httpRoutes = append(httpRoutes, httpRoute)
		i++
	}

	return httpRoutes, nil
}

// convertHTTPRules converts Ingress HTTP paths to HTTPRoute rules
func (c *Converter) convertHTTPRules(ing *networkingv1.Ingress, paths []networkingv1.HTTPIngressPath) ([]gatewayv1.HTTPRouteRule, error) {
	var rules []gatewayv1.HTTPRouteRule

	// Deduplicate paths by path+backend combination
	seen := make(map[string]bool)

	for _, path := range paths {
		key := fmt.Sprintf("%s:%s", path.Path, path.Backend.Service.Name)
		if seen[key] {
			continue
		}
		seen[key] = true

		rule := gatewayv1.HTTPRouteRule{}

		// Path match
		pathType := gatewayv1.PathMatchPathPrefix
		if path.PathType != nil && *path.PathType == networkingv1.PathTypeExact {
			pathType = gatewayv1.PathMatchExact
		}

		pathValue := path.Path
		if pathValue == "" {
			pathValue = "/"
		}

		rule.Matches = []gatewayv1.HTTPRouteMatch{
			{
				Path: &gatewayv1.HTTPPathMatch{
					Type:  &pathType,
					Value: &pathValue,
				},
			},
		}

		// Backend refs
		port := gatewayv1.PortNumber(path.Backend.Service.Port.Number)
		weight := int32(1)

		rule.BackendRefs = []gatewayv1.HTTPBackendRef{
			{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(path.Backend.Service.Name),
						Port: &port,
					},
					Weight: &weight,
				},
			},
		}

		// Apply filters from annotations
		filters, err := c.extractFilters(ing)
		if err != nil {
			return nil, err
		}
		if len(filters) > 0 {
			rule.Filters = filters
		}

		// Apply timeouts from annotations
		timeouts := c.extractTimeouts(ing)
		if timeouts != nil {
			rule.Timeouts = timeouts
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// extractFilters extracts HTTPRoute filters from annotations
func (c *Converter) extractFilters(ing *networkingv1.Ingress) ([]gatewayv1.HTTPRouteFilter, error) {
	var filters []gatewayv1.HTTPRouteFilter

	// URL Rewrite
	if rewriteTarget, exists := ing.Annotations["nginx.ingress.kubernetes.io/rewrite-target"]; exists {
		filterType := gatewayv1.HTTPRouteFilterURLRewrite
		filters = append(filters, gatewayv1.HTTPRouteFilter{
			Type: filterType,
			URLRewrite: &gatewayv1.HTTPURLRewriteFilter{
				Path: &gatewayv1.HTTPPathModifier{
					Type:               gatewayv1.FullPathHTTPPathModifier,
					ReplaceFullPath:    &rewriteTarget,
				},
			},
		})
	}

	// Redirect
	if redirect, exists := ing.Annotations["nginx.ingress.kubernetes.io/permanent-redirect"]; exists {
		filterType := gatewayv1.HTTPRouteFilterRequestRedirect
		statusCode := 301
		filters = append(filters, gatewayv1.HTTPRouteFilter{
			Type: filterType,
			RequestRedirect: &gatewayv1.HTTPRequestRedirectFilter{
				Hostname:   (*gatewayv1.PreciseHostname)(&redirect),
				StatusCode: &statusCode,
			},
		})
	}

	return filters, nil
}

// extractTimeouts extracts timeout configuration from annotations
func (c *Converter) extractTimeouts(ing *networkingv1.Ingress) *gatewayv1.HTTPRouteTimeouts {
	var timeouts *gatewayv1.HTTPRouteTimeouts

	// Check for proxy-read-timeout
	if readTimeout, exists := ing.Annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"]; exists {
		if seconds, err := strconv.Atoi(readTimeout); err == nil {
			duration := gatewayv1.Duration(fmt.Sprintf("%ds", seconds))
			if timeouts == nil {
				timeouts = &gatewayv1.HTTPRouteTimeouts{}
			}
			// Set both timeouts for complete timeout control
			timeouts.Request = &duration
			timeouts.BackendRequest = &duration
		}
	}

	// Check for proxy-send-timeout (also applies to backend)
	if sendTimeout, exists := ing.Annotations["nginx.ingress.kubernetes.io/proxy-send-timeout"]; exists {
		if seconds, err := strconv.Atoi(sendTimeout); err == nil {
			duration := gatewayv1.Duration(fmt.Sprintf("%ds", seconds))
			if timeouts == nil {
				timeouts = &gatewayv1.HTTPRouteTimeouts{}
			}
			// Only set if not already set
			if timeouts.BackendRequest == nil {
				timeouts.BackendRequest = &duration
			}
			if timeouts.Request == nil {
				timeouts.Request = &duration
			}
		}
	}

	return timeouts
}

// createDefaultBackendRule creates a rule for default backend
func (c *Converter) createDefaultBackendRule(ing *networkingv1.Ingress, backend *networkingv1.IngressBackend) gatewayv1.HTTPRouteRule {
	port := gatewayv1.PortNumber(backend.Service.Port.Number)
	weight := int32(1)
	pathValue := "/"
	pathType := gatewayv1.PathMatchPathPrefix

	return gatewayv1.HTTPRouteRule{
		Matches: []gatewayv1.HTTPRouteMatch{
			{
				Path: &gatewayv1.HTTPPathMatch{
					Type:  &pathType,
					Value: &pathValue,
				},
			},
		},
		BackendRefs: []gatewayv1.HTTPBackendRef{
			{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(backend.Service.Name),
						Port: &port,
					},
					Weight: &weight,
				},
			},
		},
	}
}

// deriveGatewayName derives Gateway name from Ingress class
func (c *Converter) deriveGatewayName(ing *networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return fmt.Sprintf("gateway-%s", *ing.Spec.IngressClassName)
	}
	if class, exists := ing.Annotations["kubernetes.io/ingress.class"]; exists {
		return fmt.Sprintf("gateway-%s", class)
	}
	return "gateway-nginx"
}

// extractHostPattern extracts pattern from hostname
func (c *Converter) extractHostPattern(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return host
}

// sanitizeName sanitizes name for Kubernetes resource
func sanitizeName(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	sanitized := reg.ReplaceAllString(strings.ToLower(name), "-")
	sanitized = strings.Trim(sanitized, "-")
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
	}
	return sanitized
}

// WriteOutput writes HTTPRoutes to output
func (c *Converter) WriteOutput(httpRoutes []interface{}, w io.Writer) error {
	for i, route := range httpRoutes {
		if i > 0 {
			fmt.Fprintln(w, "---")
		}

		var data []byte
		var err error

		if c.opts.OutputFormat == "json" {
			data, err = yaml.Marshal(route)
			if err != nil {
				return fmt.Errorf("failed to marshal HTTPRoute: %w", err)
			}
		} else {
			data, err = yaml.Marshal(route)
			if err != nil {
				return fmt.Errorf("failed to marshal HTTPRoute: %w", err)
			}
		}

		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}
