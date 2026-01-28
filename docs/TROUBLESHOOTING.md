# Troubleshooting Guide

Common issues and solutions when migrating from Ingress-NGINX to Gateway API.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Conversion Issues](#conversion-issues)
- [Validation Errors](#validation-errors)
- [Runtime Issues](#runtime-issues)
- [Performance Issues](#performance-issues)
- [Getting Help](#getting-help)

## Installation Issues

### Problem: "no such command: ingress-to-gateway"

**Symptoms**:
```bash
$ ingress-to-gateway
bash: ingress-to-gateway: command not found
```

**Solutions**:

1. **Verify installation**:
```bash
which ingress-to-gateway
# If empty, not in PATH
```

2. **Check GOPATH**:
```bash
ls $GOPATH/bin/ | grep ingress-to-gateway
# Or
ls $HOME/go/bin/ | grep ingress-to-gateway
```

3. **Add to PATH**:
```bash
export PATH=$PATH:$GOPATH/bin
# Or
export PATH=$PATH:$HOME/go/bin

# Make permanent
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
```

4. **Reinstall**:
```bash
go install github.com/mayens/ingress-to-gateway@latest
```

### Problem: "failed to create kubernetes client"

**Symptoms**:
```
Error: failed to create kubernetes client: no kubeconfig found
```

**Solutions**:

1. **Specify kubeconfig**:
```bash
ingress-to-gateway audit --kubeconfig ~/.kube/config
```

2. **Set KUBECONFIG environment variable**:
```bash
export KUBECONFIG=~/.kube/config
ingress-to-gateway audit
```

3. **Verify kubeconfig**:
```bash
kubectl cluster-info
# If this fails, fix kubectl first
```

4. **Check permissions**:
```bash
ls -l ~/.kube/config
chmod 600 ~/.kube/config
```

### Problem: Gateway API CRDs not found

**Symptoms**:
```
Error: no matches for kind "HTTPRoute" in version "gateway.networking.k8s.io/v1"
```

**Solutions**:

1. **Install Gateway API CRDs**:
```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
```

2. **Verify CRDs installed**:
```bash
kubectl get crd | grep gateway
# Should show:
# gateways.gateway.networking.k8s.io
# httproutes.gateway.networking.k8s.io
# gatewayclasses.gateway.networking.k8s.io
```

3. **Check version**:
```bash
kubectl get crd httproutes.gateway.networking.k8s.io -o yaml | grep version
# Should be v1 or later
```

## Conversion Issues

### Problem: "failed to convert ingress: not found"

**Symptoms**:
```bash
$ ingress-to-gateway convert my-ingress -n default
Error: failed to get ingress: ingresses.networking.k8s.io "my-ingress" not found
```

**Solutions**:

1. **List available Ingress resources**:
```bash
kubectl get ingress -n default
kubectl get ingress --all-namespaces
```

2. **Check namespace**:
```bash
# Use correct namespace
ingress-to-gateway convert my-ingress -n correct-namespace
```

3. **Convert from file instead**:
```bash
kubectl get ingress my-ingress -n default -o yaml > ingress.yaml
ingress-to-gateway convert -f ingress.yaml
```

### Problem: "invalid split mode"

**Symptoms**:
```
Error: invalid split mode: per-namespace
```

**Solutions**:

Valid split modes are:
- `single` (default)
- `per-host`
- `per-pattern`

```bash
# Use valid mode
ingress-to-gateway convert my-ingress --split-mode=per-host
```

### Problem: Generated HTTPRoute has no rules

**Symptoms**:
```yaml
spec:
  hostnames:
  - app.example.com
  rules: []
```

**Cause**: Ingress has no HTTP rules defined

**Solutions**:

1. **Check original Ingress**:
```bash
kubectl get ingress my-ingress -o yaml
# Verify it has rules section
```

2. **Ensure rules exist**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
spec:
  rules:
  - host: app.example.com
    http:  # <- Must have http section
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app-service
            port:
              number: 80
```

### Problem: Annotations not converted

**Symptoms**:
Generated HTTPRoute missing filters/timeouts despite Ingress having annotations

**Diagnosis**:
```bash
# Check which annotations are present
kubectl get ingress my-ingress -o yaml | grep annotations -A 20

# Check audit report
ingress-to-gateway audit -n default --detailed
```

**Solutions**:

1. **Supported annotations**: See [ANNOTATION-MAPPING.md](ANNOTATION-MAPPING.md)

2. **Unsupported annotations require manual configuration**:
```yaml
# Example: CORS not in HTTPRoute, needs gateway policy
apiVersion: gateway.nginx.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: cors-policy
spec:
  targetRef:
    kind: HTTPRoute
    name: my-httproute
  cors:
    allowOrigins:
    - "*"
```

3. **Configuration snippets** require manual review:
```yaml
# NGINX snippets cannot be automatically converted
metadata:
  annotations:
    nginx.ingress.kubernetes.io/configuration-snippet: |
      # Manual migration required
```

## Validation Errors

### Problem: "backendRequest must be <= request"

**Symptoms**:
```
❌ Errors in default/my-httproute:
  - rules[0].timeouts.backendRequest (600s) must be <= request (300s)
```

**Solution**:
```yaml
# Fix timeout constraint
# Option 1: Increase request timeout
timeouts:
  request: 600s          # Increased
  backendRequest: 600s

# Option 2: Decrease backend timeout
timeouts:
  request: 300s
  backendRequest: 300s   # Decreased
```

### Problem: "metadata.name is required"

**Symptoms**:
```
❌ Errors in /my-httproute:
  - metadata.name is required
```

**Cause**: Generated or manually created HTTPRoute is missing name

**Solution**:
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: my-httproute  # Add this
  namespace: default
spec:
  # ...
```

### Problem: "invalid hostname format"

**Symptoms**:
```
❌ Errors in default/my-httproute:
  - invalid hostname format: APP.EXAMPLE.COM
```

**Cause**: Hostnames must be lowercase

**Solution**:
```yaml
spec:
  hostnames:
  - "app.example.com"  # ✅ Lowercase
  # NOT: "APP.EXAMPLE.COM"
```

### Problem: "at least one parentRef is required"

**Symptoms**:
```
❌ Errors in default/my-httproute:
  - at least one parentRef is required
```

**Solution**:
```yaml
spec:
  parentRefs:  # Add this
  - name: gateway-nginx
  hostnames:
  - app.example.com
```

### Problem: "path.value must start with '/'"

**Symptoms**:
```
❌ Errors in default/my-httproute:
  - rules[0].matches[0].path.value must start with '/'
```

**Solution**:
```yaml
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/api"  # ✅ Starts with /
      # NOT: "api"
```

## Runtime Issues

### Problem: HTTPRoute not accepted by Gateway

**Symptoms**:
```bash
$ kubectl get httproute my-httproute
NAME           HOSTNAMES             AGE
my-httproute   ["app.example.com"]   5m

$ kubectl describe httproute my-httproute
Status:
  Parents:
    Conditions:
      Type: Accepted
      Status: False
      Reason: NoMatchingListenerHostname
```

**Diagnosis**:
```bash
# Check Gateway listeners
kubectl get gateway my-gateway -o yaml | grep -A 10 listeners
```

**Solutions**:

1. **Hostname mismatch**:
```yaml
# Gateway listener
listeners:
- hostname: "*.prod.example.com"  # Only matches *.prod.example.com

# HTTPRoute hostname
hostnames:
- "app.example.com"  # ❌ Doesn't match

# Fix: Update HTTPRoute or Gateway
hostnames:
- "app.prod.example.com"  # ✅ Matches
```

2. **Protocol mismatch**:
```yaml
# Gateway has only HTTPS listener
listeners:
- name: https
  protocol: HTTPS

# But HTTPRoute references HTTP listener
parentRefs:
- name: my-gateway
  sectionName: http  # ❌ No such listener

# Fix: Reference correct listener
parentRefs:
- name: my-gateway
  sectionName: https  # ✅ Correct
```

3. **Namespace mismatch**:
```yaml
# Gateway in gateway-system namespace
# HTTPRoute in default namespace
# But Gateway doesn't allow cross-namespace refs

# Solution: Allow cross-namespace
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
spec:
  listeners:
  - name: http
    allowedRoutes:
      namespaces:
        from: All  # or Selector
```

### Problem: 404 Not Found after migration

**Symptoms**:
```bash
$ curl -H "Host: app.example.com" http://gateway-ip/
404 Not Found
```

**Diagnosis**:
```bash
# Check HTTPRoute status
kubectl describe httproute my-httproute

# Check Gateway status
kubectl describe gateway my-gateway

# Check path matches
kubectl get httproute my-httproute -o yaml | grep -A 10 matches
```

**Solutions**:

1. **Path mismatch**:
```yaml
# Original Ingress
paths:
- path: /app
  pathType: Prefix

# Ensure HTTPRoute has same
matches:
- path:
    type: PathPrefix
    value: "/app"  # Must match exactly
```

2. **Hostname not matching**:
```bash
# Test with different hostname
curl -H "Host: app.example.com" http://gateway-ip/
curl -H "Host: www.app.example.com" http://gateway-ip/

# Check HTTPRoute hostnames
kubectl get httproute -o yaml | grep hostnames -A 3
```

3. **Backend service not found**:
```bash
# Check if service exists
kubectl get service my-service -n default

# Check HTTPRoute backendRefs
kubectl get httproute my-httproute -o yaml | grep -A 5 backendRefs
```

### Problem: 502 Bad Gateway

**Symptoms**:
```bash
$ curl -H "Host: app.example.com" http://gateway-ip/
502 Bad Gateway
```

**Diagnosis**:
```bash
# Check backend service
kubectl get service my-service -n default
kubectl get endpoints my-service -n default

# Check pods
kubectl get pods -l app=my-service

# Check Gateway logs
kubectl logs -n gateway-system -l app=gateway --tail=100
```

**Solutions**:

1. **Backend pods not ready**:
```bash
kubectl get pods -l app=my-service
# If not ready, fix pod issues
kubectl describe pod <pod-name>
```

2. **Wrong backend port**:
```yaml
# HTTPRoute specifies wrong port
backendRefs:
- name: my-service
  port: 8080  # ❌ Service listens on 80

# Fix: Use correct port
backendRefs:
- name: my-service
  port: 80  # ✅ Correct
```

3. **Service selector mismatch**:
```bash
# Check service endpoints
kubectl describe service my-service
# If no endpoints, selector doesn't match pods
```

### Problem: 504 Gateway Timeout

**Symptoms**:
```bash
$ curl -H "Host: app.example.com" http://gateway-ip/slow-endpoint
504 Gateway Timeout
```

**Diagnosis**:
```bash
# Check timeout configuration
kubectl get httproute my-httproute -o yaml | grep -A 5 timeouts

# Check backend response time
kubectl logs -n backend-ns deploy/backend | grep duration
```

**Solutions**:

1. **Increase timeouts**:
```yaml
rules:
- timeouts:
    request: 900s          # Increase from 600s
    backendRequest: 900s
```

2. **Fix slow backend** (preferred):
```bash
# Optimize backend instead of just increasing timeouts
# Check backend logs for slow queries, etc.
```

See [TIMEOUT-CONFIGURATION.md](TIMEOUT-CONFIGURATION.md) for detailed timeout guide.

### Problem: TLS certificate errors

**Symptoms**:
```bash
$ curl https://app.example.com
curl: (60) SSL certificate problem: unable to get local issuer certificate
```

**Diagnosis**:
```bash
# Check Gateway TLS configuration
kubectl get gateway my-gateway -o yaml | grep -A 10 tls

# Check certificate secret
kubectl get secret my-tls-cert -n default
kubectl describe secret my-tls-cert
```

**Solutions**:

1. **Secret not found**:
```bash
# Create TLS secret
kubectl create secret tls my-tls-cert \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key
```

2. **Wrong secret name in Gateway**:
```yaml
listeners:
- tls:
    certificateRefs:
    - name: my-tls-cert  # Must match secret name
```

3. **Certificate expired**:
```bash
# Check certificate expiry
kubectl get secret my-tls-cert -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -dates
```

## Performance Issues

### Problem: High latency after migration

**Symptoms**:
Response times increased after switching to Gateway API

**Diagnosis**:
```bash
# Compare latencies
# (Use your monitoring tools)

# Check Gateway resource usage
kubectl top pod -n gateway-system

# Check Gateway logs for slow requests
kubectl logs -n gateway-system -l app=gateway | grep "duration"
```

**Solutions**:

1. **Gateway under-provisioned**:
```yaml
# Increase Gateway resources
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway
spec:
  template:
    spec:
      containers:
      - name: gateway
        resources:
          requests:
            cpu: 2000m      # Increased
            memory: 2Gi     # Increased
```

2. **Too many HTTPRoutes**:
```bash
# Consolidate HTTPRoutes
# Use single mode instead of per-host
ingress-to-gateway convert my-ingress --split-mode=single
```

3. **Gateway configuration**:
```yaml
# Tune Gateway settings (gateway-specific)
# Example for NGINX Gateway
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-config
data:
  worker-processes: "auto"
  worker-connections: "10240"
```

### Problem: Gateway pods crashing

**Symptoms**:
```bash
$ kubectl get pods -n gateway-system
NAME            READY   STATUS             RESTARTS
gateway-xxx     0/1     CrashLoopBackOff   5
```

**Diagnosis**:
```bash
# Check pod logs
kubectl logs -n gateway-system gateway-xxx --previous

# Check events
kubectl describe pod -n gateway-system gateway-xxx

# Check resource usage
kubectl top pod -n gateway-system
```

**Solutions**:

1. **OOMKilled**:
```yaml
# Increase memory limit
resources:
  limits:
    memory: 4Gi  # Increased from 2Gi
```

2. **Configuration error**:
```bash
# Check Gateway configuration
kubectl get gateway -o yaml
# Fix any configuration errors
```

## Getting Help

### Before Asking for Help

1. **Check logs**:
```bash
# Gateway logs
kubectl logs -n gateway-system -l app=gateway --tail=100

# HTTPRoute status
kubectl describe httproute <name> -n <namespace>

# Gateway status
kubectl describe gateway <name> -n <namespace>
```

2. **Run audit**:
```bash
ingress-to-gateway audit --detailed > audit-report.txt
```

3. **Validate HTTPRoute**:
```bash
ingress-to-gateway validate my-httproute.yaml
```

4. **Check versions**:
```bash
# Tool version
ingress-to-gateway --version

# Gateway API version
kubectl get crd httproutes.gateway.networking.k8s.io -o yaml | grep version

# Kubernetes version
kubectl version
```

### Where to Get Help

1. **GitHub Issues**:
   - https://github.com/mayens/ingress-to-gateway/issues
   - Include: audit report, HTTPRoute YAML, error messages, logs

2. **GitHub Discussions**:
   - https://github.com/mayens/ingress-to-gateway/discussions
   - For questions, best practices, migration strategies

3. **Gateway API Slack**:
   - https://kubernetes.slack.com/archives/CR0H13KGA
   - For Gateway API-specific questions

### Information to Include

When asking for help, provide:

1. **Version information**:
```bash
ingress-to-gateway --version
kubectl version
kubectl get crd httproutes.gateway.networking.k8s.io -o jsonpath='{.spec.versions[*].name}'
```

2. **Audit report**:
```bash
ingress-to-gateway audit --detailed > audit.txt
```

3. **Original Ingress**:
```bash
kubectl get ingress <name> -n <namespace> -o yaml > original-ingress.yaml
```

4. **Generated HTTPRoute**:
```bash
cat generated-httproute.yaml
```

5. **Error messages**:
```bash
# Exact error output
# Gateway logs
# HTTPRoute status
```

6. **What you've tried**:
- List troubleshooting steps already attempted
- Include commands run and their output

## Quick Reference

### Common Commands

```bash
# Audit
ingress-to-gateway audit --all-namespaces --detailed

# Convert
ingress-to-gateway convert <ingress-name> -n <namespace>

# Validate
ingress-to-gateway validate <httproute-file>

# Check HTTPRoute status
kubectl get httproute -A
kubectl describe httproute <name> -n <namespace>

# Check Gateway status
kubectl get gateway -A
kubectl describe gateway <name> -n <namespace>

# Check Gateway logs
kubectl logs -n gateway-system -l app=gateway --tail=100

# Test endpoint
curl -v -H "Host: <hostname>" http://<gateway-ip><path>
```

### Status Checks

```bash
# Is HTTPRoute accepted?
kubectl get httproute <name> -o jsonpath='{.status.parents[0].conditions[?(@.type=="Accepted")].status}'

# Is Gateway ready?
kubectl get gateway <name> -o jsonpath='{.status.conditions[?(@.type=="Programmed")].status}'

# Are backends available?
kubectl get endpoints <service-name>

# Gateway external IP
kubectl get gateway <name> -o jsonpath='{.status.addresses[0].value}'
```

## Summary

✅ Check tool installation and PATH
✅ Verify kubeconfig and cluster access
✅ Ensure Gateway API CRDs installed
✅ Validate HTTPRoute before applying
✅ Check HTTPRoute acceptance status
✅ Verify Gateway and backend status
✅ Review timeout configuration
✅ Monitor Gateway logs
✅ Include full context when asking for help

## Next Steps

- Review [Getting Started Guide](GETTING-STARTED.md) for step-by-step migration
- Check [Annotation Mapping](ANNOTATION-MAPPING.md) for feature equivalents
- See [Migration Strategies](MIGRATION-STRATEGIES.md) for phased approaches
- Read [Timeout Configuration](TIMEOUT-CONFIGURATION.md) for timeout best practices
