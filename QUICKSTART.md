# Quick Start Guide

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/mayens/ingress-to-gateway.git
cd ingress-to-gateway

# Build the binary
make build

# Install to $GOPATH/bin
make install
```

### Using go install

```bash
go install github.com/mayens/ingress-to-gateway@latest
```

## Basic Usage

### 1. Audit Your Ingress Resources

Start by auditing your existing Ingress resources to understand migration readiness:

```bash
# Audit all Ingress in current namespace
ingress-to-gateway audit

# Audit across all namespaces
ingress-to-gateway audit --all-namespaces

# Generate detailed report with recommendations
ingress-to-gateway audit --detailed
```

**Example Output:**
```
╔═══════════════════════════════════════════════════════════════════════════════╗
║                      INGRESS MIGRATION AUDIT REPORT                           ║
╚═══════════════════════════════════════════════════════════════════════════════╝

Total Ingress Resources: 5

Migration Readiness Summary:
  ✅ READY: 3
  ⚠️  MOSTLY_READY: 1
  ❌ MANUAL_REVIEW_REQUIRED: 1

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
INGRESS DETAILS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Ingress: default/my-app-ingress
────────────────────────────────────────────────────────────────────────────────
  Ingress Class: nginx
  Hosts: 2 | Paths: 3 | TLS: true
  Hostnames: app.example.com, api.example.com
  Migration Readiness: ✅ READY (Complexity: 8)
  Detected Features: URL_REWRITE, TLS_TERMINATION, PROXY_READ_TIMEOUT
  💡 Recommendations:
    • Use 'single' split mode (default) for optimal Gateway API resource usage
    • Both timeouts.request and timeouts.backendRequest will be set for complete timeout control
    • Ensure Gateway has matching HTTPS listeners configured
```

### 2. Convert a Single Ingress

Convert an Ingress resource to HTTPRoute:

```bash
# Convert from cluster
ingress-to-gateway convert my-ingress -n default

# Convert from file
ingress-to-gateway convert -f ingress.yaml

# Save to file
ingress-to-gateway convert my-ingress -n default -o httproute.yaml

# Use per-host splitting strategy
ingress-to-gateway convert my-ingress --split-mode=per-host
```

**Split Modes:**
- `single` (default): One HTTPRoute for all hostnames - optimal for most cases
- `per-host`: Separate HTTPRoute per hostname - maximum flexibility
- `per-pattern`: Grouped by hostname patterns - intelligent organization

### 3. Batch Convert Multiple Ingresses

Convert all Ingress resources in a namespace:

```bash
# Convert all in current namespace
ingress-to-gateway batch -o ./httproutes

# Convert across all namespaces
ingress-to-gateway batch --all-namespaces -o ./httproutes

# With custom split mode
ingress-to-gateway batch --split-mode=per-host -o ./httproutes
```

**Output Structure:**
```
httproutes/
├── default/
│   ├── my-app-ingress-httproute.yaml
│   └── api-ingress-httproute.yaml
├── production/
│   ├── web-ingress-httproute.yaml
│   └── backend-ingress-httproute.yaml
└── staging/
    └── test-ingress-httproute.yaml
```

### 4. Validate HTTPRoute

Validate your generated HTTPRoute before applying:

```bash
# Validate a file
ingress-to-gateway validate httproute.yaml

# Strict mode (treat warnings as errors)
ingress-to-gateway validate httproute.yaml --strict
```

**Example Output:**
```
✅ Validation passed: No issues found
```

Or with issues:
```
❌ Errors in default/my-httproute:
  - rules[0].backendRefs[0].port is required

⚠️  Warnings in default/my-httproute:
  - path /api appears in multiple rules: [0 2]
```

## Common Workflows

### Complete Migration Workflow

```bash
# Step 1: Audit to understand what you have
ingress-to-gateway audit --all-namespaces --detailed > audit-report.txt

# Step 2: Batch convert all resources
ingress-to-gateway batch --all-namespaces -o ./httproutes

# Step 3: Validate each generated HTTPRoute
for file in httproutes/**/*.yaml; do
  ingress-to-gateway validate "$file" || echo "Failed: $file"
done

# Step 4: Review and apply to cluster
kubectl apply -f httproutes/
```

### Convert Specific Ingress with Custom Options

```bash
# Convert with specific gateway reference
ingress-to-gateway convert my-ingress \
  --gateway=my-custom-gateway \
  --gateway-class=nginx \
  -o httproute.yaml

# Validate
ingress-to-gateway validate httproute.yaml

# Apply to cluster
kubectl apply -f httproute.yaml
```

## Supported NGINX Annotations

ingress-to-gateway supports 17+ NGINX Ingress annotations:

### Routing & Rewrite
- ✅ `nginx.ingress.kubernetes.io/rewrite-target` → URLRewrite filter
- ✅ `nginx.ingress.kubernetes.io/app-root` → RequestRedirect filter

### Redirects
- ✅ `nginx.ingress.kubernetes.io/ssl-redirect` → RequestRedirect filter
- ✅ `nginx.ingress.kubernetes.io/permanent-redirect` → RequestRedirect filter
- ✅ `nginx.ingress.kubernetes.io/temporal-redirect` → RequestRedirect filter

### Timeouts
- ✅ `nginx.ingress.kubernetes.io/proxy-read-timeout` → timeouts.backendRequest
- ✅ `nginx.ingress.kubernetes.io/proxy-send-timeout` → timeouts.backendRequest
- ✅ Both annotations → timeouts.request AND timeouts.backendRequest

### Canary Deployments
- ✅ `nginx.ingress.kubernetes.io/canary` → Traffic splitting
- ✅ `nginx.ingress.kubernetes.io/canary-weight` → backendRefs.weight
- ✅ `nginx.ingress.kubernetes.io/canary-by-header` → Header matches

### Backend & TLS
- ✅ `nginx.ingress.kubernetes.io/backend-protocol` → Backend configuration
- ✅ TLS termination → Gateway HTTPS listeners

### Other
- ✅ `nginx.ingress.kubernetes.io/proxy-body-size`
- ✅ CORS annotations
- ✅ Authentication annotations
- ⚠️  Custom snippets (requires manual review)

## Output Examples

### Simple Ingress → HTTPRoute

**Input Ingress:**
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
spec:
  ingressClassName: nginx
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app-service
            port:
              number: 80
```

**Generated HTTPRoute:**
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: example-httproute
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

## Comparison with ingress2gateway

| Feature | ingress-to-gateway | ingress2gateway |
|---------|-------------------|-----------------|
| NGINX Annotations | **17+** | 5 |
| Audit Capability | ✅ Yes | ❌ No |
| Split Strategies | ✅ 3 modes | ❌ No |
| Timeout Support | ✅ Both fields | ⚠️ Partial |
| Validation | ✅ Built-in | ❌ No |
| Batch Conversion | ✅ Yes | ⚠️ Basic |
| Documentation | ✅ Comprehensive | ⚠️ Basic |
| Rule Deduplication | ✅ Yes | ❌ No |

## Next Steps

1. Check out [examples/](examples/) for more sample Ingress resources
2. Read [MIGRATION-GUIDE.md](../audit-ingress/MIGRATION-TABLE.md) for detailed annotation mappings
3. Review [CONTRIBUTING.md](CONTRIBUTING.md) if you want to contribute
4. Open an issue if you encounter any problems

## Tips

- **Always audit first**: Understand complexity before converting
- **Validate before applying**: Catch issues early
- **Use single mode by default**: It's the most efficient for Gateway API
- **Test in staging**: Validate behavior before production
- **Monitor timeouts**: Both request and backendRequest timeouts are set for complete control

## Troubleshooting

### "no kubeconfig found"
Set the `--kubeconfig` flag or `KUBECONFIG` environment variable:
```bash
ingress-to-gateway audit --kubeconfig ~/.kube/config
```

### "validation failed"
Check the error messages and fix issues in the HTTPRoute:
```bash
ingress-to-gateway validate httproute.yaml
```

### Complex annotations not converted
Some annotations (like custom snippets) require manual review:
```bash
# Check audit report for blockers
ingress-to-gateway audit --detailed
```
