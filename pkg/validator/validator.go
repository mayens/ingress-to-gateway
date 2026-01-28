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

package validator

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/yaml"
)

// Validator validates HTTPRoute resources
type Validator struct {
	strict bool
}

// ValidationResult contains validation results for a resource
type ValidationResult struct {
	ResourceName string
	Errors       []string
	Warnings     []string
}

// NewValidator creates a new Validator
func NewValidator(strict bool) *Validator {
	return &Validator{
		strict: strict,
	}
}

// ValidateFile validates HTTPRoute resources in a file
func (v *Validator) ValidateFile(ctx context.Context, path string) ([]*ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Split by YAML document separator
	docs := strings.Split(string(data), "---")
	var results []*ValidationResult

	for i, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var httpRoute gatewayv1.HTTPRoute
		if err := yaml.Unmarshal([]byte(doc), &httpRoute); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document %d: %w", i+1, err)
		}

		result := v.validateHTTPRoute(&httpRoute)
		results = append(results, result)
	}

	return results, nil
}

// validateHTTPRoute validates a single HTTPRoute
func (v *Validator) validateHTTPRoute(hr *gatewayv1.HTTPRoute) *ValidationResult {
	result := &ValidationResult{
		ResourceName: fmt.Sprintf("%s/%s", hr.Namespace, hr.Name),
	}

	// Validate metadata
	v.validateMetadata(hr, result)

	// Validate hostnames
	v.validateHostnames(hr, result)

	// Validate parent refs
	v.validateParentRefs(hr, result)

	// Validate rules
	v.validateRules(hr, result)

	return result
}

// validateMetadata validates HTTPRoute metadata
func (v *Validator) validateMetadata(hr *gatewayv1.HTTPRoute, result *ValidationResult) {
	if hr.Name == "" {
		result.Errors = append(result.Errors, "metadata.name is required")
	} else {
		// Validate name format
		nameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
		if !nameRegex.MatchString(hr.Name) {
			result.Errors = append(result.Errors, "metadata.name must consist of lower case alphanumeric characters or '-'")
		}
		if len(hr.Name) > 63 {
			result.Errors = append(result.Errors, "metadata.name must be no more than 63 characters")
		}
	}

	if hr.Namespace == "" {
		result.Warnings = append(result.Warnings, "metadata.namespace not specified, will use default namespace")
	}
}

// validateHostnames validates HTTPRoute hostnames
func (v *Validator) validateHostnames(hr *gatewayv1.HTTPRoute, result *ValidationResult) {
	if len(hr.Spec.Hostnames) == 0 {
		result.Warnings = append(result.Warnings, "no hostnames specified, HTTPRoute will match all hostnames")
	}

	for _, hostname := range hr.Spec.Hostnames {
		// Basic hostname validation
		if string(hostname) == "" {
			result.Errors = append(result.Errors, "empty hostname not allowed")
			continue
		}

		// Check for valid hostname format
		hostnameRegex := regexp.MustCompile(`^(\*\.)?[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
		if !hostnameRegex.MatchString(string(hostname)) {
			result.Errors = append(result.Errors, fmt.Sprintf("invalid hostname format: %s", hostname))
		}
	}
}

// validateParentRefs validates parent references
func (v *Validator) validateParentRefs(hr *gatewayv1.HTTPRoute, result *ValidationResult) {
	if len(hr.Spec.ParentRefs) == 0 {
		result.Errors = append(result.Errors, "at least one parentRef is required")
		return
	}

	for i, ref := range hr.Spec.ParentRefs {
		if ref.Name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("parentRefs[%d].name is required", i))
		}

		// Validate Gateway kind
		if ref.Kind != nil && *ref.Kind != "Gateway" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("parentRefs[%d].kind is %s, expected Gateway", i, *ref.Kind))
		}
	}
}

// validateRules validates HTTPRoute rules
func (v *Validator) validateRules(hr *gatewayv1.HTTPRoute, result *ValidationResult) {
	if len(hr.Spec.Rules) == 0 {
		result.Errors = append(result.Errors, "at least one rule is required")
		return
	}

	for i, rule := range hr.Spec.Rules {
		// Validate matches
		if len(rule.Matches) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("rules[%d]: no matches specified, will match all requests", i))
		}

		for j, match := range rule.Matches {
			v.validateMatch(&match, i, j, result)
		}

		// Validate backend refs
		if len(rule.BackendRefs) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("rules[%d]: at least one backendRef is required", i))
		}

		for j, backendRef := range rule.BackendRefs {
			v.validateBackendRef(&backendRef, i, j, result)
		}

		// Validate timeouts
		if rule.Timeouts != nil {
			v.validateTimeouts(rule.Timeouts, i, result)
		}

		// Validate filters
		for j, filter := range rule.Filters {
			v.validateFilter(&filter, i, j, result)
		}
	}

	// Check for path conflicts
	v.checkPathConflicts(hr, result)
}

// validateMatch validates an HTTP route match
func (v *Validator) validateMatch(match *gatewayv1.HTTPRouteMatch, ruleIdx, matchIdx int, result *ValidationResult) {
	if match.Path != nil {
		if match.Path.Value == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].matches[%d].path.value is required", ruleIdx, matchIdx))
		} else {
			// Validate path format
			if !strings.HasPrefix(*match.Path.Value, "/") {
				result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].matches[%d].path.value must start with '/'", ruleIdx, matchIdx))
			}
		}
	}
}

// validateBackendRef validates a backend reference
func (v *Validator) validateBackendRef(ref *gatewayv1.HTTPBackendRef, ruleIdx, refIdx int, result *ValidationResult) {
	if ref.Name == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].backendRefs[%d].name is required", ruleIdx, refIdx))
	}

	if ref.Port == nil {
		result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].backendRefs[%d].port is required", ruleIdx, refIdx))
	}

	if ref.Weight != nil && *ref.Weight < 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].backendRefs[%d].weight must be >= 0", ruleIdx, refIdx))
	}
}

// validateTimeouts validates timeout configuration
func (v *Validator) validateTimeouts(timeouts *gatewayv1.HTTPRouteTimeouts, ruleIdx int, result *ValidationResult) {
	// Validate duration format and constraint: backendRequest <= request
	if timeouts.Request != nil && timeouts.BackendRequest != nil {
		requestSec := parseDuration(string(*timeouts.Request))
		backendSec := parseDuration(string(*timeouts.BackendRequest))

		if requestSec > 0 && backendSec > 0 && backendSec > requestSec {
			result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].timeouts.backendRequest (%s) must be <= request (%s)", ruleIdx, *timeouts.BackendRequest, *timeouts.Request))
		}
	}

	// Validate duration format
	if timeouts.Request != nil && !isValidDuration(string(*timeouts.Request)) {
		result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].timeouts.request: invalid duration format", ruleIdx))
	}
	if timeouts.BackendRequest != nil && !isValidDuration(string(*timeouts.BackendRequest)) {
		result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].timeouts.backendRequest: invalid duration format", ruleIdx))
	}
}

// validateFilter validates a route filter
func (v *Validator) validateFilter(filter *gatewayv1.HTTPRouteFilter, ruleIdx, filterIdx int, result *ValidationResult) {
	switch filter.Type {
	case gatewayv1.HTTPRouteFilterURLRewrite:
		if filter.URLRewrite == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].filters[%d]: URLRewrite is required for type URLRewrite", ruleIdx, filterIdx))
		}
	case gatewayv1.HTTPRouteFilterRequestRedirect:
		if filter.RequestRedirect == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("rules[%d].filters[%d]: RequestRedirect is required for type RequestRedirect", ruleIdx, filterIdx))
		}
	}
}

// checkPathConflicts checks for conflicting path matches
func (v *Validator) checkPathConflicts(hr *gatewayv1.HTTPRoute, result *ValidationResult) {
	paths := make(map[string][]int)

	for i, rule := range hr.Spec.Rules {
		for _, match := range rule.Matches {
			if match.Path != nil && match.Path.Value != nil {
				key := fmt.Sprintf("%s:%s", *match.Path.Type, *match.Path.Value)
				paths[key] = append(paths[key], i)
			}
		}
	}

	for path, ruleIndices := range paths {
		if len(ruleIndices) > 1 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("path %s appears in multiple rules: %v", path, ruleIndices))
		}
	}
}

// isValidDuration checks if a duration string is valid
func isValidDuration(duration string) bool {
	durationRegex := regexp.MustCompile(`^[0-9]+(h|m|s|ms)$`)
	return durationRegex.MatchString(duration)
}

// parseDuration parses a duration string to seconds
func parseDuration(duration string) int {
	durationRegex := regexp.MustCompile(`^([0-9]+)(h|m|s|ms)$`)
	matches := durationRegex.FindStringSubmatch(duration)
	if len(matches) != 3 {
		return 0
	}

	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "h":
		return value * 3600
	case "m":
		return value * 60
	case "s":
		return value
	case "ms":
		return value / 1000
	default:
		return 0
	}
}
