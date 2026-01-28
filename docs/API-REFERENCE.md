# API Reference

Complete command-line interface reference for ingress-to-gateway.

## Table of Contents

- [Global Flags](#global-flags)
- [Commands](#commands)
  - [audit](#audit)
  - [convert](#convert)
  - [batch](#batch)
  - [validate](#validate)
  - [completion](#completion)
- [Exit Codes](#exit-codes)
- [Configuration File](#configuration-file)
- [Environment Variables](#environment-variables)

## Global Flags

These flags are available for all commands:

### `--config` string

Path to configuration file.

**Default**: `$HOME/.ingress-to-gateway.yaml`

**Example**:
```bash
ingress-to-gateway audit --config /path/to/config.yaml
```

### `--kubeconfig` string

Path to kubeconfig file for cluster access.

**Default**: Uses standard kubeconfig resolution:
1. `--kubeconfig` flag
2. `$KUBECONFIG` environment variable
3. `$HOME/.kube/config`
4. In-cluster config (if running in pod)

**Example**:
```bash
ingress-to-gateway audit --kubeconfig ~/.kube/production-config
```

### `-n, --namespace` string

Kubernetes namespace to operate in.

**Default**: Current namespace from kubeconfig context

**Example**:
```bash
ingress-to-gateway audit -n production
ingress-to-gateway convert my-ingress --namespace staging
```

### `-h, --help`

Display help information for any command.

**Example**:
```bash
ingress-to-gateway --help
ingress-to-gateway audit --help
```

## Commands

### audit

Audit Ingress resources for Gateway API migration readiness.

#### Synopsis

```bash
ingress-to-gateway audit [flags]
```

#### Description

Analyzes Ingress resources and generates a comprehensive report showing:
- Migration readiness assessment
- Annotation usage
- Complexity scores
- Potential issues and blockers
- Migration recommendations

#### Flags

##### `-A, --all-namespaces`

Audit Ingress resources across all namespaces.

**Default**: `false`

**Example**:
```bash
ingress-to-gateway audit --all-namespaces
ingress-to-gateway audit -A
```

##### `-d, --detailed`

Generate detailed report with recommendations.

**Default**: `false`

**Example**:
```bash
ingress-to-gateway audit --detailed
ingress-to-gateway audit -d
```

##### `-o, --output` string

Output format for the report.

**Valid values**: `table`, `json`, `yaml`

**Default**: `table`

**Examples**:
```bash
# Table format (default, human-readable)
ingress-to-gateway audit --output table

# JSON format (machine-readable)
ingress-to-gateway audit --output json > audit.json

# YAML format
ingress-to-gateway audit --output yaml > audit.yaml
```

#### Examples

**Basic audit of current namespace**:
```bash
ingress-to-gateway audit
```

**Audit all namespaces with details**:
```bash
ingress-to-gateway audit --all-namespaces --detailed
```

**Generate JSON report for automation**:
```bash
ingress-to-gateway audit -A -o json | jq '.[] | select(.MigrationReadiness=="READY")'
```

**Save detailed audit to file**:
```bash
ingress-to-gateway audit --all-namespaces --detailed > audit-report-$(date +%Y%m%d).txt
```

#### Output Format

**Table Format** (default):
```
╔═══════════════════════════════════════════════════════════════════════════════╗
║                      INGRESS MIGRATION AUDIT REPORT                           ║
╚═══════════════════════════════════════════════════════════════════════════════╝

Total Ingress Resources: 3

Migration Readiness Summary:
  ✅ READY: 2
  ⚠️  MOSTLY_READY: 1

📋 Ingress: default/my-app-ingress
────────────────────────────────────────────────────────────────────────────────
  Ingress Class: nginx
  Hosts: 2 | Paths: 3 | TLS: true
  Migration Readiness: ✅ READY (Complexity: 8)
```

**JSON Format**:
```json
[
  {
    "Name": "my-app-ingress",
    "Namespace": "default",
    "IngressClass": "nginx",
    "HostCount": 2,
    "Hostnames": ["app.example.com", "api.example.com"],
    "PathCount": 3,
    "TLSEnabled": true,
    "DetectedFeatures": ["URL_REWRITE", "TLS_TERMINATION", "PROXY_READ_TIMEOUT"],
    "ComplexityScore": 8,
    "MigrationReadiness": "READY",
    "Issues": [],
    "Recommendations": [
      "Use 'single' split mode (default) for optimal Gateway API resource usage",
      "Both timeouts.request and timeouts.backendRequest will be set"
    ]
  }
]
```

---

### convert

Convert Ingress resource to Gateway API HTTPRoute.

#### Synopsis

```bash
ingress-to-gateway convert [ingress-name] [flags]
```

#### Description

Translates an Ingress resource to Gateway API HTTPRoute with support for:
- 17+ NGINX Ingress annotations
- Multiple split strategies (single, per-host, per-pattern)
- Automatic rule deduplication
- Proper timeout configuration
- TLS configuration

#### Flags

##### `-f, --file` string

Path to input file containing Ingress resource.

**When to use**: Convert Ingress from YAML file instead of cluster

**Example**:
```bash
ingress-to-gateway convert -f my-ingress.yaml
ingress-to-gateway convert --file /path/to/ingress.yaml
```

##### `-o, --output-file` string

Path to output file for HTTPRoute.

**Default**: Output to stdout

**Example**:
```bash
ingress-to-gateway convert my-ingress -n default -o my-httproute.yaml
```

##### `--split-mode` string

HTTPRoute split strategy.

**Valid values**: `single`, `per-host`, `per-pattern`

**Default**: `single`

**Descriptions**:
- `single`: One HTTPRoute for all hostnames (optimal for Gateway API)
- `per-host`: Separate HTTPRoute per hostname (maximum flexibility)
- `per-pattern`: Grouped by hostname patterns (intelligent organization)

**Examples**:
```bash
# Single HTTPRoute (default)
ingress-to-gateway convert my-ingress --split-mode=single

# One HTTPRoute per hostname
ingress-to-gateway convert my-ingress --split-mode=per-host

# Grouped by domain pattern
ingress-to-gateway convert my-ingress --split-mode=per-pattern
```

##### `--gateway` string

Gateway name to reference in parentRefs.

**Default**: Derived from Ingress class (e.g., `gateway-nginx`)

**Example**:
```bash
ingress-to-gateway convert my-ingress --gateway=my-custom-gateway
```

##### `--gateway-class` string

Gateway class name.

**Default**: `nginx`

**Example**:
```bash
ingress-to-gateway convert my-ingress --gateway-class=istio
```

##### `--format` string

Output format for HTTPRoute.

**Valid values**: `yaml`, `json`

**Default**: `yaml`

**Example**:
```bash
ingress-to-gateway convert my-ingress --format=json
```

#### Arguments

##### `ingress-name` (positional)

Name of the Ingress resource to convert.

**Required**: Yes (unless using `--file`)

**Example**:
```bash
ingress-to-gateway convert my-app-ingress -n default
```

#### Examples

**Basic conversion**:
```bash
ingress-to-gateway convert my-ingress -n default
```

**Convert and save to file**:
```bash
ingress-to-gateway convert my-ingress -n default -o httproute.yaml
```

**Convert from file**:
```bash
ingress-to-gateway convert -f ingress.yaml -o httproute.yaml
```

**Per-host splitting**:
```bash
ingress-to-gateway convert my-ingress --split-mode=per-host
```

**Custom gateway reference**:
```bash
ingress-to-gateway convert my-ingress \
  --gateway=production-gateway \
  --gateway-class=nginx \
  -o httproute.yaml
```

**JSON output**:
```bash
ingress-to-gateway convert my-ingress --format=json | jq .
```

#### Output

**YAML Format** (default):
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: my-ingress-httproute
  namespace: default
spec:
  parentRefs:
  - name: gateway-nginx
  hostnames:
  - "app.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: "/"
    timeouts:
      request: 600s
      backendRequest: 600s
    backendRefs:
    - name: app-service
      port: 80
      weight: 1
```

---

### batch

Batch convert multiple Ingress resources.

#### Synopsis

```bash
ingress-to-gateway batch [flags]
```

#### Description

Converts multiple Ingress resources to HTTPRoutes with intelligent organization:
- Groups by namespace
- Creates organized directory structure
- Applies consistent naming conventions
- Generates summary report

#### Flags

##### `-o, --output-dir` string

Output directory for HTTPRoute files.

**Default**: `./httproutes`

**Example**:
```bash
ingress-to-gateway batch -o ./migration/httproutes
ingress-to-gateway batch --output-dir /tmp/httproutes
```

##### `-A, --all-namespaces`

Convert Ingress resources across all namespaces.

**Default**: `false`

**Example**:
```bash
ingress-to-gateway batch --all-namespaces -o ./httproutes
ingress-to-gateway batch -A -o ./output
```

##### `--split-mode` string

HTTPRoute split strategy for all conversions.

**Valid values**: `single`, `per-host`, `per-pattern`

**Default**: `single`

**Example**:
```bash
ingress-to-gateway batch --split-mode=per-host -o ./httproutes
```

##### `--gateway-class` string

Gateway class name for all HTTPRoutes.

**Default**: `nginx`

**Example**:
```bash
ingress-to-gateway batch --gateway-class=istio -o ./httproutes
```

#### Examples

**Batch convert current namespace**:
```bash
ingress-to-gateway batch -o ./httproutes
```

**Batch convert all namespaces**:
```bash
ingress-to-gateway batch --all-namespaces -o ./httproutes
```

**With per-host splitting**:
```bash
ingress-to-gateway batch -A --split-mode=per-host -o ./httproutes
```

**Custom gateway class**:
```bash
ingress-to-gateway batch \
  --all-namespaces \
  --gateway-class=istio \
  -o ./httproutes
```

#### Output Structure

```
httproutes/
├── default/
│   ├── app1-ingress-httproute.yaml
│   ├── app2-ingress-httproute.yaml
│   └── api-ingress-httproute.yaml
├── production/
│   ├── web-ingress-httproute.yaml
│   └── backend-ingress-httproute.yaml
└── staging/
    └── test-ingress-httproute.yaml
```

#### Summary Output

```
Processing namespace: default
  Converting: app1-ingress
    Created: app1-ingress-httproute.yaml
  Converting: app2-ingress
    Created: app2-ingress-httproute.yaml

Processing namespace: production
  Converting: web-ingress
    Created: web-ingress-httproute.yaml

Batch conversion complete:
  Successfully converted: 3
  Failed: 0
  Output directory: ./httproutes
```

---

### validate

Validate HTTPRoute resources.

#### Synopsis

```bash
ingress-to-gateway validate [file] [flags]
```

#### Description

Validates HTTPRoute resources for:
- YAML/JSON syntax
- Gateway API schema compliance
- Reference validity (Gateway, Service)
- Timeout constraints
- Path match conflicts
- Best practice recommendations

#### Flags

##### `--strict`

Treat warnings as errors.

**Default**: `false`

**Example**:
```bash
ingress-to-gateway validate httproute.yaml --strict
```

#### Arguments

##### `file` (positional)

Path to HTTPRoute YAML/JSON file.

**Required**: Yes

**Example**:
```bash
ingress-to-gateway validate my-httproute.yaml
```

#### Examples

**Basic validation**:
```bash
ingress-to-gateway validate httproute.yaml
```

**Strict mode (warnings as errors)**:
```bash
ingress-to-gateway validate httproute.yaml --strict
```

**Validate multiple files**:
```bash
for file in httproutes/**/*.yaml; do
  ingress-to-gateway validate "$file" || exit 1
done
```

**In CI/CD pipeline**:
```bash
#!/bin/bash
set -e
ingress-to-gateway validate httproute.yaml --strict
echo "Validation passed"
```

#### Output

**Success**:
```
✅ Validation passed: No issues found
```

**With Errors**:
```
❌ Errors in default/my-httproute:
  - metadata.name is required
  - rules[0].backendRefs[0].port is required
  - rules[0].timeouts.backendRequest (600s) must be <= request (300s)
```

**With Warnings**:
```
⚠️  Warnings in default/my-httproute:
  - no hostnames specified, HTTPRoute will match all hostnames
  - path / appears in multiple rules: [0 2]

✅ Validation passed with warnings
```

**In Strict Mode** (warnings cause failure):
```
❌ Validation failed
⚠️  Warnings in default/my-httproute:
  - no hostnames specified, HTTPRoute will match all hostnames

Exit code: 1
```

---

### completion

Generate shell completion scripts.

#### Synopsis

```bash
ingress-to-gateway completion [shell]
```

#### Description

Generates shell completion scripts for bash, zsh, fish, or PowerShell.

#### Supported Shells

- `bash`
- `zsh`
- `fish`
- `powershell`

#### Examples

**Bash**:
```bash
# Generate completion script
ingress-to-gateway completion bash > /etc/bash_completion.d/ingress-to-gateway

# Or for current session
source <(ingress-to-gateway completion bash)

# Add to ~/.bashrc
echo 'source <(ingress-to-gateway completion bash)' >> ~/.bashrc
```

**Zsh**:
```bash
# Generate completion script
ingress-to-gateway completion zsh > "${fpath[1]}/_ingress-to-gateway"

# Or for current session
source <(ingress-to-gateway completion zsh)

# Add to ~/.zshrc
echo 'source <(ingress-to-gateway completion zsh)' >> ~/.zshrc
```

**Fish**:
```bash
ingress-to-gateway completion fish > ~/.config/fish/completions/ingress-to-gateway.fish
```

**PowerShell**:
```powershell
ingress-to-gateway completion powershell | Out-String | Invoke-Expression
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (conversion failed, validation failed, etc.) |
| 2 | Invalid arguments or flags |

**Examples**:

```bash
# Check exit code
ingress-to-gateway validate httproute.yaml
echo $?  # 0 if valid, 1 if invalid

# Use in scripts
if ingress-to-gateway validate httproute.yaml; then
  echo "Valid"
  kubectl apply -f httproute.yaml
else
  echo "Invalid"
  exit 1
fi
```

---

## Configuration File

### File Location

Default: `$HOME/.ingress-to-gateway.yaml`

Override: `--config /path/to/config.yaml`

### File Format

```yaml
# Kubernetes configuration
kubeconfig: ~/.kube/config
namespace: default

# Conversion defaults
convert:
  splitMode: single
  gatewayClass: nginx
  outputFormat: yaml

# Audit defaults
audit:
  allNamespaces: false
  detailed: false
  outputFormat: table

# Validation defaults
validate:
  strict: false

# Batch defaults
batch:
  outputDir: ./httproutes
  splitMode: single
  gatewayClass: nginx
```

### Example Usage

```bash
# Create config file
cat > ~/.ingress-to-gateway.yaml <<EOF
namespace: production
convert:
  splitMode: per-host
  gatewayClass: istio
audit:
  detailed: true
EOF

# Commands now use these defaults
ingress-to-gateway audit  # Uses detailed: true from config
ingress-to-gateway convert my-ingress  # Uses splitMode: per-host
```

---

## Environment Variables

### `KUBECONFIG`

Path to kubeconfig file.

**Example**:
```bash
export KUBECONFIG=~/.kube/production-config
ingress-to-gateway audit
```

### `INGRESS_TO_GATEWAY_CONFIG`

Path to configuration file (alternative to `--config`).

**Example**:
```bash
export INGRESS_TO_GATEWAY_CONFIG=/etc/ingress-to-gateway/config.yaml
ingress-to-gateway audit
```

### Priority Order

Configuration values are resolved in this order (highest priority first):

1. Command-line flags
2. Environment variables
3. Configuration file
4. Defaults

**Example**:
```bash
# Config file has: namespace: default
# Command specifies: -n production
# Result: Uses production (flag overrides config)
```

---

## Examples

### Complete Workflow

```bash
# 1. Audit all namespaces
ingress-to-gateway audit --all-namespaces --detailed > audit-report.txt

# 2. Review audit report
cat audit-report.txt | grep "READY"

# 3. Batch convert ready resources
ingress-to-gateway batch --all-namespaces -o httproutes/

# 4. Validate all generated HTTPRoutes
for file in httproutes/**/*.yaml; do
  ingress-to-gateway validate "$file" --strict || exit 1
done

# 5. Apply to cluster
kubectl apply -f httproutes/ --recursive
```

### CI/CD Integration

```bash
#!/bin/bash
# .github/workflows/validate-httproutes.sh

set -e

# Convert Ingress to HTTPRoute
ingress-to-gateway convert -f ingress.yaml -o httproute.yaml

# Validate
ingress-to-gateway validate httproute.yaml --strict

# Apply to test cluster
kubectl apply -f httproute.yaml --dry-run=server

echo "✅ HTTPRoute validation passed"
```

### Automation Script

```bash
#!/bin/bash
# migrate-namespace.sh

NAMESPACE=$1
OUTPUT_DIR="./httproutes/$NAMESPACE"

echo "Migrating namespace: $NAMESPACE"

# Audit
ingress-to-gateway audit -n "$NAMESPACE" --detailed -o json > "$NAMESPACE-audit.json"

# Check readiness
READY_COUNT=$(jq '[.[] | select(.MigrationReadiness=="READY")] | length' "$NAMESPACE-audit.json")
echo "Ready resources: $READY_COUNT"

# Convert
ingress-to-gateway batch -n "$NAMESPACE" -o "$OUTPUT_DIR"

# Validate
for file in "$OUTPUT_DIR"/*.yaml; do
  if ! ingress-to-gateway validate "$file"; then
    echo "Validation failed: $file"
    exit 1
  fi
done

echo "✅ Migration complete for $NAMESPACE"
```

---

## See Also

- [Getting Started Guide](GETTING-STARTED.md)
- [Annotation Mapping](ANNOTATION-MAPPING.md)
- [Migration Strategies](MIGRATION-STRATEGIES.md)
- [Timeout Configuration](TIMEOUT-CONFIGURATION.md)
- [Troubleshooting](TROUBLESHOOTING.md)

---

## Version

To check the version:

```bash
ingress-to-gateway version
# Or
ingress-to-gateway --version
```

Output:
```
ingress-to-gateway version 0.1.0 (commit: abc1234, built: 2026-01-28T15:30:00Z)
```
