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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestValidateHTTPRoute(t *testing.T) {
	tests := []struct {
		name       string
		httpRoute  *gatewayv1.HTTPRoute
		strict     bool
		wantErrors int
		wantWarnings int
	}{
		{
			name: "Valid HTTPRoute",
			httpRoute: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{
						{Name: "gateway-nginx"},
					},
				},
				Hostnames: []gatewayv1.Hostname{"app.example.com"},
					Rules: []gatewayv1.HTTPRouteRule{
						{
							Matches: []gatewayv1.HTTPRouteMatch{
								{
									Path: &gatewayv1.HTTPPathMatch{
										Type:  pathMatchTypePtr(gatewayv1.PathMatchPathPrefix),
										Value: stringPtr("/"),
									},
								},
							},
							BackendRefs: []gatewayv1.HTTPBackendRef{
								{
									BackendRef: gatewayv1.BackendRef{
										BackendObjectReference: gatewayv1.BackendObjectReference{
											Name: "app-service",
											Port: portNumberPtr(80),
										},
									},
								},
							},
						},
					},
				},
			},
			strict:       false,
			wantErrors:   0,
			wantWarnings: 0,
		},
		{
			name: "Missing name",
			httpRoute: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{
							{Name: "gateway-nginx"},
						},
					},
				},
			},
			strict:       false,
			wantErrors:   2, // missing name + no rules
			wantWarnings: 1, // no hostnames
		},
		{
			name: "Invalid hostname format",
			httpRoute: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{
						{Name: "gateway-nginx"},
					},
				},
				Hostnames: []gatewayv1.Hostname{"APP.EXAMPLE.COM"}, // Invalid - uppercase
				},
			},
			strict:       false,
			wantErrors:   2, // invalid hostname + no rules
			wantWarnings: 0,
		},
		{
			name: "No parent refs",
			httpRoute: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{"app.example.com"},
				},
			},
			strict:       false,
			wantErrors:   2, // no parent refs + no rules
			wantWarnings: 0,
		},
		{
			name: "No rules",
			httpRoute: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{
						{Name: "gateway-nginx"},
					},
				},
				Hostnames: []gatewayv1.Hostname{"app.example.com"},
					Rules:     []gatewayv1.HTTPRouteRule{},
				},
			},
			strict:       false,
			wantErrors:   1,
			wantWarnings: 0,
		},
		{
			name: "Invalid timeout constraint",
			httpRoute: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{
						{Name: "gateway-nginx"},
					},
				},
				Hostnames: []gatewayv1.Hostname{"app.example.com"},
					Rules: []gatewayv1.HTTPRouteRule{
						{
							Timeouts: &gatewayv1.HTTPRouteTimeouts{
								Request:        durationPtr("300s"),
								BackendRequest: durationPtr("600s"), // ERROR: backend > request
							},
							BackendRefs: []gatewayv1.HTTPBackendRef{
								{
									BackendRef: gatewayv1.BackendRef{
										BackendObjectReference: gatewayv1.BackendObjectReference{
											Name: "app-service",
											Port: portNumberPtr(80),
										},
									},
								},
							},
						},
					},
				},
			},
			strict:       false,
			wantErrors:   1,
			wantWarnings: 1, // no matches specified
		},
		{
			name: "Warning - no hostnames",
			httpRoute: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{
							{Name: "gateway-nginx"},
						},
					},
					// No hostnames specified
					Rules: []gatewayv1.HTTPRouteRule{
						{
							BackendRefs: []gatewayv1.HTTPBackendRef{
								{
									BackendRef: gatewayv1.BackendRef{
										BackendObjectReference: gatewayv1.BackendObjectReference{
											Name: "app-service",
											Port: portNumberPtr(80),
										},
									},
								},
							},
						},
					},
				},
			},
			strict:       false,
			wantErrors:   0,
			wantWarnings: 2, // no hostnames + no matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(tt.strict)
			result := v.validateHTTPRoute(tt.httpRoute)

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("validateHTTPRoute() errors = %v, want %v. Errors: %v", len(result.Errors), tt.wantErrors, result.Errors)
			}

			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("validateHTTPRoute() warnings = %v, want %v. Warnings: %v", len(result.Warnings), tt.wantWarnings, result.Warnings)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name       string
		path       *gatewayv1.HTTPPathMatch
		wantErrors int
	}{
		{
			name: "Valid path",
			path: &gatewayv1.HTTPPathMatch{
				Type:  pathMatchTypePtr(gatewayv1.PathMatchPathPrefix),
				Value: stringPtr("/api"),
			},
			wantErrors: 0,
		},
		{
			name: "Invalid path - no leading slash",
			path: &gatewayv1.HTTPPathMatch{
				Type:  pathMatchTypePtr(gatewayv1.PathMatchPathPrefix),
				Value: stringPtr("api"),
			},
			wantErrors: 1,
		},
		{
			name: "Missing value",
			path: &gatewayv1.HTTPPathMatch{
				Type: pathMatchTypePtr(gatewayv1.PathMatchPathPrefix),
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(false)
			result := &ValidationResult{ResourceName: "test"}
			v.validateMatch(&gatewayv1.HTTPRouteMatch{Path: tt.path}, 0, 0, result)

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("validatePath() errors = %v, want %v", len(result.Errors), tt.wantErrors)
			}
		})
	}
}

func TestIsValidDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     bool
	}{
		{"Valid seconds", "60s", true},
		{"Valid minutes", "5m", true},
		{"Valid hours", "2h", true},
		{"Valid milliseconds", "100ms", true},
		{"Invalid - no unit", "60", false},
		{"Invalid - wrong unit", "60x", false},
		{"Invalid - negative", "-60s", false},
		{"Invalid - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDuration(tt.duration)
			if got != tt.want {
				t.Errorf("isValidDuration(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     int
	}{
		{"Seconds", "60s", 60},
		{"Minutes", "5m", 300},
		{"Hours", "2h", 7200},
		{"Milliseconds", "1000ms", 1},
		{"Invalid", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDuration(tt.duration)
			if got != tt.want {
				t.Errorf("parseDuration(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}

func TestValidateTimeouts(t *testing.T) {
	tests := []struct {
		name       string
		timeouts   *gatewayv1.HTTPRouteTimeouts
		wantErrors int
	}{
		{
			name: "Valid - backend <= request",
			timeouts: &gatewayv1.HTTPRouteTimeouts{
				Request:        durationPtr("600s"),
				BackendRequest: durationPtr("600s"),
			},
			wantErrors: 0,
		},
		{
			name: "Valid - backend < request",
			timeouts: &gatewayv1.HTTPRouteTimeouts{
				Request:        durationPtr("600s"),
				BackendRequest: durationPtr("300s"),
			},
			wantErrors: 0,
		},
		{
			name: "Invalid - backend > request",
			timeouts: &gatewayv1.HTTPRouteTimeouts{
				Request:        durationPtr("300s"),
				BackendRequest: durationPtr("600s"),
			},
			wantErrors: 1,
		},
		{
			name: "Invalid duration format",
			timeouts: &gatewayv1.HTTPRouteTimeouts{
				Request: durationPtr("invalid"),
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(false)
			result := &ValidationResult{ResourceName: "test"}
			v.validateTimeouts(tt.timeouts, 0, result)

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("validateTimeouts() errors = %v, want %v. Errors: %v", len(result.Errors), tt.wantErrors, result.Errors)
			}
		})
	}
}

func TestValidateBackendRef(t *testing.T) {
	tests := []struct {
		name       string
		ref        *gatewayv1.HTTPBackendRef
		wantErrors int
	}{
		{
			name: "Valid backend ref",
			ref: &gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: "app-service",
						Port: portNumberPtr(80),
					},
					Weight: int32Ptr(1),
				},
			},
			wantErrors: 0,
		},
		{
			name: "Missing name",
			ref: &gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Port: portNumberPtr(80),
					},
				},
			},
			wantErrors: 1,
		},
		{
			name: "Missing port",
			ref: &gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: "app-service",
					},
				},
			},
			wantErrors: 1,
		},
		{
			name: "Negative weight",
			ref: &gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: "app-service",
						Port: portNumberPtr(80),
					},
					Weight: int32Ptr(-1),
				},
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(false)
			result := &ValidationResult{ResourceName: "test"}
			v.validateBackendRef(tt.ref, 0, 0, result)

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("validateBackendRef() errors = %v, want %v. Errors: %v", len(result.Errors), tt.wantErrors, result.Errors)
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func portNumberPtr(p int) *gatewayv1.PortNumber {
	pn := gatewayv1.PortNumber(p)
	return &pn
}

func int32Ptr(i int32) *int32 {
	return &i
}

func pathMatchTypePtr(t gatewayv1.PathMatchType) *gatewayv1.PathMatchType {
	return &t
}

func durationPtr(d string) *gatewayv1.Duration {
	duration := gatewayv1.Duration(d)
	return &duration
}
