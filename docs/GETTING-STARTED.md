# Getting Started Guide

Complete guide to get started with ingress-to-gateway for migrating from Ingress-NGINX to Gateway API.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Your First Migration](#your-first-migration)
- [Understanding the Workflow](#understanding-the-workflow)
- [Common Scenarios](#common-scenarios)
- [Best Practices](#best-practices)

## Prerequisites

### Required

- **Kubernetes Cluster**: v1.25 or later
- **Gateway API CRDs**: Installed in your cluster
- **kubectl**: Configured to access your cluster
- **Go**: 1.21+ (if building from source)

### Installing Gateway API CRDs

```bash
# Install Gateway API CRDs (v1.0.0)
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml

# Verify installation
kubectl get crd gateways.gateway.networking.k8s.io
kubectl get crd httproutes.gateway.networking.k8s.io
```

### Installing a Gateway Controller

You need a Gateway API controller. For NGINX:

```bash
# Install NGINX Gateway Fabric
kubectl apply -f https://github.com/nginxinc/nginx-gateway-fabric/releases/latest/download/crds.yaml
kubectl apply -f https://github.com/nginxinc/nginx-gateway-fabric/releases/latest/download/nginx-gateway.yaml
```

## Installation

### Option 1: Pre-built Binary (Recommended)

Download from releases page:

```bash
# Linux
wget https://github.com/mayens/ingress-to-gateway/releases/download/v0.1.0/ingress-to-gateway-linux-amd64
chmod +x ingress-to-gateway-linux-amd64
sudo mv ingress-to-gateway-linux-amd64 /usr/local/bin/ingress-to-gateway

# macOS
wget https://github.com/mayens/ingress-to-gateway/releases/download/v0.1.0/ingress-to-gateway-darwin-amd64
chmod +x ingress-to-gateway-darwin-amd64
sudo mv ingress-to-gateway-darwin-amd64 /usr/local/bin/ingress-to-gateway

# Verify installation
ingress-to-gateway --version
```

### Option 2: Using go install

```bash
go install github.com/mayens/ingress-to-gateway@latest

# Binary will be in $GOPATH/bin or $HOME/go/bin
```

### Option 3: Build from Source

```bash
# Clone repository
git clone https://github.com/mayens/ingress-to-gateway.git
cd ingress-to-gateway

# Build
make build

# Install
sudo cp bin/ingress-to-gateway /usr/local/bin/

# Or install to GOPATH
make install
```

## Your First Migration

Let's walk through a complete migration step by step.

### Step 1: Explore Your Current Setup

First, check what Ingress resources you have:

```bash
# List all Ingress resources
kubectl get ingress --all-namespaces

# Example output:
# NAMESPACE   NAME              CLASS   HOSTS                ADDRESS         PORTS
# default     my-app-ingress    nginx   app.example.com      10.0.0.100      80, 443
# default     api-ingress       nginx   api.example.com      10.0.0.100      80, 443
# staging     test-ingress      nginx   test.example.com     10.0.0.101      80
```

### Step 2: Run an Audit

Audit your Ingress resources to understand migration complexity:

```bash
# Audit all Ingress in current namespace
ingress-to-gateway audit

# Audit across all namespaces with detailed report
ingress-to-gateway audit --all-namespaces --detailed
```

**Example Output:**

```
╔═══════════════════════════════════════════════════════════════════════════════╗
║                      INGRESS MIGRATION AUDIT REPORT                           ║
╚═══════════════════════════════════════════════════════════════════════════════╝

Total Ingress Resources: 3

Migration Readiness Summary:
  ✅ READY: 2
  ⚠️  MOSTLY_READY: 1

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
    • Both timeouts.request and timeouts.backendRequest will be set
    • Ensure Gateway has matching HTTPS listeners configured
```

### Step 3: Create or Verify Gateway

Before converting, ensure you have a Gateway resource:

```bash
# Check existing Gateways
kubectl get gateway -n default

# If none exist, create one
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-nginx
  namespace: default
spec:
  gatewayClassName: nginx
  listeners:
  - name: http
    protocol: HTTP
    port: 80
  - name: https
    protocol: HTTPS
    port: 443
    tls:
      mode: Terminate
      certificateRefs:
      - name: example-tls
EOF
```

### Step 4: Convert Single Ingress

Convert your first Ingress to HTTPRoute:

```bash
# Convert and output to stdout
ingress-to-gateway convert my-app-ingress -n default

# Convert and save to file
ingress-to-gateway convert my-app-ingress -n default -o my-app-httproute.yaml

# Review the generated HTTPRoute
cat my-app-httproute.yaml
```

**Example Generated HTTPRoute:**

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: my-app-ingress-httproute
  namespace: default
spec:
  parentRefs:
  - name: gateway-nginx
  hostnames:
  - "app.example.com"
  - "api.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: "/"
    timeouts:
      request: 600s
      backendRequest: 600s
    backendRefs:
    - name: my-app-service
      port: 80
      weight: 1
```

### Step 5: Validate Before Applying

Always validate before applying to your cluster:

```bash
# Validate the generated HTTPRoute
ingress-to-gateway validate my-app-httproute.yaml

# Expected output if valid:
# ✅ Validation passed: No issues found
```

### Step 6: Test in Non-Production First

Apply to a test/staging environment first:

```bash
# Apply the HTTPRoute
kubectl apply -f my-app-httproute.yaml

# Check HTTPRoute status
kubectl get httproute my-app-ingress-httproute -n default

# Check if it's accepted by the Gateway
kubectl describe httproute my-app-ingress-httproute -n default
```

### Step 7: Test Traffic

Verify traffic flows through the new HTTPRoute:

```bash
# Test HTTP endpoint
curl -H "Host: app.example.com" http://<gateway-ip>/

# Test HTTPS endpoint
curl -H "Host: app.example.com" https://<gateway-ip>/

# Compare with original Ingress
curl -H "Host: app.example.com" http://<ingress-ip>/
```

### Step 8: Monitor and Compare

Monitor both for a period (e.g., 24-48 hours):

```bash
# Watch HTTPRoute status
kubectl get httproute -n default -w

# Check Gateway logs
kubectl logs -n <gateway-namespace> -l app=gateway --tail=100 -f

# Compare metrics between Ingress and HTTPRoute
```

### Step 9: Cutover

Once confident, switch traffic:

```bash
# Option 1: Delete old Ingress (HTTPRoute takes over)
kubectl delete ingress my-app-ingress -n default

# Option 2: Modify DNS to point to Gateway IP
# (Update your DNS records)

# Option 3: Update Service to point to Gateway
# (If using LoadBalancer services)
```

### Step 10: Cleanup

After successful migration:

```bash
# Remove old Ingress resources
kubectl delete ingress --all -n default

# Optionally remove old Ingress Controller
kubectl delete -f <old-ingress-controller-manifest.yaml>
```

## Understanding the Workflow

### Migration Phases

```
┌─────────────┐
│   Phase 1   │  Discovery & Assessment
│   AUDIT     │  • Identify all Ingress resources
└──────┬──────┘  • Assess complexity
       │         • Identify blockers
       ▼
┌─────────────┐
│   Phase 2   │  Preparation
│   PLAN      │  • Create Gateway resources
└──────┬──────┘  • Review annotation mappings
       │         • Plan split strategy
       ▼
┌─────────────┐
│   Phase 3   │  Conversion
│  CONVERT    │  • Generate HTTPRoutes
└──────┬──────┘  • Validate output
       │         • Review configurations
       ▼
┌─────────────┐
│   Phase 4   │  Validation
│  VALIDATE   │  • Syntax validation
└──────┬──────┘  • Schema compliance
       │         • Reference checks
       ▼
┌─────────────┐
│   Phase 5   │  Testing
│    TEST     │  • Apply to staging
└──────┬──────┘  • Traffic validation
       │         • Performance testing
       ▼
┌─────────────┐
│   Phase 6   │  Deployment
│   DEPLOY    │  • Apply to production
└──────┬──────┘  • Monitor
       │         • Rollback if needed
       ▼
┌─────────────┐
│   Phase 7   │  Cleanup
│  CLEANUP    │  • Remove old Ingress
└─────────────┘  • Update documentation
```

## Common Scenarios

### Scenario 1: Simple Web Application

**Requirements:**
- Single hostname
- Single backend service
- TLS termination

```bash
# Audit
ingress-to-gateway audit -n default

# Convert
ingress-to-gateway convert simple-app-ingress -n default -o simple-app.yaml

# Validate
ingress-to-gateway validate simple-app.yaml

# Apply
kubectl apply -f simple-app.yaml
```

### Scenario 2: Microservices with Multiple Paths

**Requirements:**
- Multiple path-based routes
- Different backend services
- Timeouts configured

```bash
# Convert with default single mode (optimal)
ingress-to-gateway convert microservices-ingress -n default -o microservices.yaml

# The tool automatically:
# - Deduplicates rules
# - Sets proper timeouts
# - Maintains path ordering
```

### Scenario 3: Multi-Tenant Application

**Requirements:**
- Many hostnames (10+)
- Different backends per hostname
- Need separate management

```bash
# Use per-host split mode
ingress-to-gateway convert multi-tenant-ingress \
  --split-mode=per-host \
  -n default \
  -o multi-tenant/

# This creates separate HTTPRoute per hostname
ls multi-tenant/
# multi-tenant-ingress-httproute-1.yaml
# multi-tenant-ingress-httproute-2.yaml
# ... (one per hostname)
```

### Scenario 4: Batch Migration

**Requirements:**
- Migrate entire namespace
- Organized output
- Consistent naming

```bash
# Batch convert all Ingress in namespace
ingress-to-gateway batch -n production -o ./httproutes

# Output structure:
# httproutes/
# └── production/
#     ├── app1-httproute.yaml
#     ├── app2-httproute.yaml
#     └── app3-httproute.yaml

# Validate all
for file in httproutes/production/*.yaml; do
  ingress-to-gateway validate "$file"
done

# Apply all
kubectl apply -f httproutes/production/
```

### Scenario 5: Canary Deployment

**Requirements:**
- Canary annotations present
- Traffic splitting needed

```bash
# The tool automatically converts canary annotations
ingress-to-gateway convert canary-ingress -n default

# Result: HTTPRoute with weighted backend refs
# backendRefs:
# - name: stable-service
#   port: 80
#   weight: 90
# - name: canary-service
#   port: 80
#   weight: 10
```

## Best Practices

### 1. Always Start with Audit

```bash
# Get the full picture before converting
ingress-to-gateway audit --all-namespaces --detailed > audit-report.txt

# Review the report
cat audit-report.txt | grep "MANUAL_REVIEW_REQUIRED"
```

### 2. Use Version Control

```bash
# Save generated HTTPRoutes to Git
mkdir -p k8s/httproutes
ingress-to-gateway batch --all-namespaces -o k8s/httproutes
cd k8s
git add httproutes/
git commit -m "Generated HTTPRoutes from Ingress resources"
```

### 3. Test in Isolation

```bash
# Create test namespace
kubectl create namespace migration-test

# Copy Ingress to test namespace
kubectl get ingress my-app -n production -o yaml | \
  sed 's/namespace: production/namespace: migration-test/' | \
  kubectl apply -f -

# Convert in test namespace
ingress-to-gateway convert my-app -n migration-test -o test-httproute.yaml

# Apply and test
kubectl apply -f test-httproute.yaml
```

### 4. Validate Everything

```bash
# Create validation script
cat > validate-all.sh <<'EOF'
#!/bin/bash
for file in $(find . -name "*httproute.yaml"); do
  echo "Validating $file..."
  if ! ingress-to-gateway validate "$file"; then
    echo "❌ Validation failed: $file"
    exit 1
  fi
done
echo "✅ All HTTPRoutes valid"
EOF

chmod +x validate-all.sh
./validate-all.sh
```

### 5. Monitor During Migration

```bash
# Watch HTTPRoute status
watch kubectl get httproute --all-namespaces

# Monitor Gateway
kubectl logs -n gateway-system -l app=gateway -f

# Compare traffic
# (Use your monitoring tools to compare metrics)
```

### 6. Document Your Migration

Create a migration log:

```bash
cat > MIGRATION-LOG.md <<EOF
# Migration Log

## Date: $(date)

## Pre-migration Audit
$(ingress-to-gateway audit --all-namespaces --detailed)

## Resources Migrated
- [ ] Ingress: default/app1
- [ ] Ingress: default/app2
- [ ] Ingress: production/api

## Issues Encountered
- None

## Rollback Plan
\`\`\`bash
kubectl apply -f backup/ingress-backup.yaml
\`\`\`
EOF
```

### 7. Keep Backups

```bash
# Backup all Ingress resources before migration
kubectl get ingress --all-namespaces -o yaml > ingress-backup-$(date +%Y%m%d).yaml

# Backup in case you need to rollback
tar czf ingress-backup-$(date +%Y%m%d).tar.gz ingress-backup-*.yaml
```

## Next Steps

Now that you've completed your first migration:

1. Read [Complete Annotation Mapping](ANNOTATION-MAPPING.md) for detailed conversion rules
2. Review [Migration Strategies](MIGRATION-STRATEGIES.md) for different approaches
3. Check [Timeout Configuration](TIMEOUT-CONFIGURATION.md) for timeout best practices
4. Consult [Troubleshooting](TROUBLESHOOTING.md) if you encounter issues

## Getting Help

- **Issues**: https://github.com/mayens/ingress-to-gateway/issues
- **Discussions**: https://github.com/mayens/ingress-to-gateway/discussions
- **Documentation**: https://github.com/mayens/ingress-to-gateway/tree/main/docs

## Summary

✅ Install ingress-to-gateway
✅ Run audit to assess complexity
✅ Create Gateway resources
✅ Convert Ingress to HTTPRoute
✅ Validate generated resources
✅ Test in non-production
✅ Monitor and compare
✅ Cutover to HTTPRoute
✅ Cleanup old resources

You're now ready to migrate from Ingress to Gateway API!
