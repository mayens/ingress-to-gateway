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

package version

var (
	// Version is the current version of the tool
	Version = "0.1.0"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is the build date
	BuildDate = "unknown"
)

// Info returns version information
func Info() string {
	return Version
}

// Full returns full version information
func Full() string {
	return "ingress-to-gateway version " + Version + " (commit: " + GitCommit + ", built: " + BuildDate + ")"
}
