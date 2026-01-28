# Timeout Configuration

Comprehensive guide to understanding and configuring timeouts when migrating from Ingress-NGINX to Gateway API.

## Table of Contents

- [Overview](#overview)
- [NGINX Ingress Timeouts](#nginx-ingress-timeouts)
- [Gateway API Timeouts](#gateway-api-timeouts)
- [Mapping Guide](#mapping-guide)
- [Best Practices](#best-practices)
- [Common Scenarios](#common-scenarios)
- [Troubleshooting](#troubleshooting)

## Overview

Timeout configuration is one of the most critical aspects of migration. Gateway API introduces a more structured timeout model compared to NGINX Ingress.

### Key Differences

| Aspect | NGINX Ingress | Gateway API |
|--------|--------------|-------------|
| Granularity | Multiple timeout annotations | Two timeout fields |
| Scope | Backend-focused | Request lifecycle |
| Constraint | Independent values | backendRequest ≤ request |
| Default | NGINX defaults | No defaults (inherit from Gateway) |

## NGINX Ingress Timeouts

### Available Timeout Annotations

#### 1. `proxy-read-timeout`

**What it controls**: Time to wait for backend response after sending request

**Default**: 60s

**When it matters**:
- Slow backend processing
- Large file downloads
- Long-running API calls

**Example**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
```

#### 2. `proxy-send-timeout`

**What it controls**: Time to wait for backend to accept request body

**Default**: 60s

**When it matters**:
- Large file uploads
- Slow-accepting backends
- High latency networks

**Example**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
```

#### 3. `proxy-connect-timeout`

**What it controls**: Time to establish TCP connection to backend

**Default**: 60s

**When it matters**:
- Backend startup time
- Network issues
- Connection pool exhaustion

**Example**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "10"
```

#### 4. `client-body-timeout`

**What it controls**: Time to wait for client request body

**Default**: 60s

**When it matters**:
- Client uploads
- Slow clients
- Mobile networks

**Example**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/client-body-timeout: "300"
```

### NGINX Timeout Flow

```
Client → NGINX → Backend

client-body-timeout
    ├─ Time for client to send request body
    │
proxy-connect-timeout
    ├─ Time to connect to backend
    │
proxy-send-timeout
    ├─ Time for backend to accept request
    │
proxy-read-timeout
    └─ Time for backend to respond
```

## Gateway API Timeouts

Gateway API simplifies timeouts into two main fields:

### 1. `request` Timeout

**Scope**: **Client → Gateway → Backend → Gateway → Client**

**What it controls**: Total end-to-end request time

**When it matters**:
- Overall SLA requirements
- Preventing hung requests
- Gateway processing delays

**Default**: Inherited from Gateway (often 60s)

**Example**:
```yaml
rules:
- timeouts:
    request: 600s
  backendRefs:
  - name: app-service
    port: 80
```

### 2. `backendRequest` Timeout

**Scope**: **Gateway → Backend → Gateway**

**What it controls**: Time for backend round-trip only

**When it matters**:
- Backend-specific timeouts
- Isolating backend delays
- Different backends, different timeouts

**Default**: Inherited from Gateway (often 30s)

**Example**:
```yaml
rules:
- timeouts:
    backendRequest: 300s
  backendRefs:
  - name: backend-service
    port: 8080
```

### Gateway API Timeout Constraint

**Critical Rule**: `backendRequest ≤ request`

```yaml
# ✅ Valid
timeouts:
  request: 600s
  backendRequest: 300s

# ✅ Valid (equal is ok)
timeouts:
  request: 600s
  backendRequest: 600s

# ❌ Invalid
timeouts:
  request: 300s
  backendRequest: 600s  # ERROR: backend > request
```

### Gateway API Timeout Flow

```
Client → Gateway → Backend → Gateway → Client

request (total timeout)
    ├─ Client to Gateway
    │
    backendRequest (backend-only timeout)
    │   ├─ Gateway to Backend connection
    │   ├─ Backend processing
    │   └─ Backend to Gateway response
    │
    └─ Gateway to Client response
```

## Mapping Guide

### Mapping NGINX to Gateway API

#### Scenario 1: Only `proxy-read-timeout`

**NGINX Configuration**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
```

**Gateway API Configuration**:
```yaml
rules:
- timeouts:
    request: 600s          # Total timeout
    backendRequest: 600s   # Backend timeout
```

**Reasoning**:
- `proxy-read-timeout` is primarily about backend response time
- Set both fields to maintain behavior
- Protects against both backend AND gateway delays

#### Scenario 2: Both `proxy-read-timeout` and `proxy-send-timeout`

**NGINX Configuration**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
```

**Gateway API Configuration**:
```yaml
rules:
- timeouts:
    request: 600s          # Total timeout
    backendRequest: 600s   # Backend timeout
```

**Reasoning**:
- Both NGINX timeouts relate to backend communication
- Use the maximum value if different
- Set both Gateway API fields

#### Scenario 3: Different Timeout Values

**NGINX Configuration**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
```

**Gateway API Configuration**:
```yaml
rules:
- timeouts:
    request: 600s          # Use max(300, 600)
    backendRequest: 600s   # Use max(300, 600)
```

**Reasoning**:
- Use the maximum value to avoid prematurely terminating requests
- Ensures no existing working requests break

#### Scenario 4: `proxy-connect-timeout` Present

**NGINX Configuration**:
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "10"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
```

**Gateway API Configuration**:
```yaml
rules:
- timeouts:
    request: 610s          # read + connect
    backendRequest: 610s   # read + connect
```

**Reasoning**:
- Connect timeout is part of backend request time
- Add connect + read for total backend timeout
- Though Gateway API doesn't expose separate connect timeout

#### Scenario 5: No Timeout Annotations

**NGINX Configuration**:
```yaml
# No timeout annotations
```

**Gateway API Configuration**:
```yaml
rules:
- timeouts:
    request: 60s           # Default: 60s
    backendRequest: 60s    # Default: 60s
```

**Reasoning**:
- Use Gateway defaults (typically 60s)
- Or explicitly set to NGINX defaults
- Verify Gateway default timeouts first

### Why Set Both Fields?

#### Question: "If NGINX only has backend timeouts, why set `request` timeout?"

**Answer**: For complete timeout control and behavior matching.

1. **Gateway Processing Time**: Gateway itself takes time (routing, TLS, filters)
   ```
   Client request → [Gateway: 50ms] → Backend → [Gateway: 50ms] → Client response
   ```
   If only `backendRequest` is set, gateway processing time is unbounded.

2. **Network Delays**: Between client and gateway
   ```
   Client [slow network: 30s] → Gateway → Backend [fast: 2s]
   ```
   Without `request` timeout, client delays are unbounded.

3. **Behavior Match**: NGINX Ingress has implicit total timeout
   - NGINX waits for client → processes → waits for backend → responds
   - All these steps have timeouts
   - Gateway API `request` timeout matches this behavior

4. **Observability**: Different timeout triggers indicate different problems
   - `backendRequest` timeout → Backend is slow
   - `request` timeout (but not backend) → Network or gateway issue

#### Example: Backend Timeout vs Request Timeout

**Scenario A: Backend timeout**
```yaml
timeouts:
  request: 600s
  backendRequest: 300s
```

```
Timeline:
0s    → Client sends request
50ms  → Gateway receives, forwards to backend
50ms  → Backend starts processing
350ms → Backend still processing
        ❌ backendRequest timeout (300s exceeded)
        Gateway returns 504 Gateway Timeout
```

**Scenario B: Request timeout (Gateway delay)**
```yaml
timeouts:
  request: 300s
  backendRequest: 600s
```

```
Timeline:
0s    → Client sends request
50ms  → Gateway receives
100ms → Gateway stuck (CPU saturation, complex routing)
...
310s  → Gateway still processing
        ❌ request timeout (300s exceeded)
        Gateway returns 504 Gateway Timeout
```

Having both timeouts helps diagnose which component is slow.

## Best Practices

### 1. Always Set Both Timeout Fields

```yaml
# ✅ Good: Both timeouts set
rules:
- timeouts:
    request: 600s
    backendRequest: 600s

# ⚠️ Okay but not recommended: Only backend timeout
rules:
- timeouts:
    backendRequest: 600s

# ❌ Incomplete: Only request timeout
rules:
- timeouts:
    request: 600s
```

### 2. Use Same Value for Both (Usually)

```yaml
# ✅ Good: Same value (simplest)
timeouts:
  request: 600s
  backendRequest: 600s

# ✅ Good: Different if you have reason
timeouts:
  request: 660s          # Backend + 60s buffer for gateway
  backendRequest: 600s
```

### 3. Consider Your Application

**Short API calls**:
```yaml
timeouts:
  request: 30s
  backendRequest: 30s
```

**Long-running operations**:
```yaml
timeouts:
  request: 1800s  # 30 minutes
  backendRequest: 1800s
```

**File uploads/downloads**:
```yaml
timeouts:
  request: 3600s  # 1 hour
  backendRequest: 3600s
```

**Websockets/SSE** (requires different approach):
```yaml
# HTTPRoute doesn't work well for long-lived connections
# Consider using TCPRoute or GRPCRoute
```

### 4. Document Why

```yaml
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/api/reports"
  timeouts:
    # Reports can take up to 10 minutes to generate
    request: 600s
    backendRequest: 600s
  backendRefs:
  - name: report-service
    port: 8080
```

### 5. Test Timeout Behavior

```bash
# Test backend timeout
curl -H "Host: app.example.com" http://gateway-ip/slow-endpoint

# Test with timeout
timeout 300s curl -H "Host: app.example.com" http://gateway-ip/slow-endpoint

# Verify you get 504 Gateway Timeout
```

## Common Scenarios

### Scenario 1: Microservices API

**Requirements**:
- Fast response times
- 99% requests < 5s
- Maximum 30s for slow clients

**Configuration**:
```yaml
rules:
- timeouts:
    request: 30s
    backendRequest: 30s
  backendRefs:
  - name: api-service
    port: 8080
```

### Scenario 2: File Upload Service

**Requirements**:
- Large files (up to 10GB)
- Slow clients (mobile)
- Processing time: ~1 hour

**Configuration**:
```yaml
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/upload"
  timeouts:
    request: 7200s        # 2 hours
    backendRequest: 7200s
  backendRefs:
  - name: upload-service
    port: 8080
```

### Scenario 3: Batch Processing API

**Requirements**:
- Submit job: < 5s
- Long-running: up to 6 hours
- Use async pattern

**Configuration**:
```yaml
rules:
# Submit endpoint (fast)
- matches:
  - path:
      type: Exact
      value: "/api/batch"
    method: POST
  timeouts:
    request: 30s
    backendRequest: 30s
  backendRefs:
  - name: batch-service
    port: 8080

# Status endpoint (fast)
- matches:
  - path:
      type: PathPrefix
      value: "/api/batch/"
    method: GET
  timeouts:
    request: 10s
    backendRequest: 10s
  backendRefs:
  - name: batch-service
    port: 8080
```

**Note**: Don't use synchronous HTTP for long operations. Use async patterns.

### Scenario 4: Legacy Application

**Requirements**:
- Existing timeouts: proxy-read-timeout: 600
- Don't know exact requirements
- Want to match existing behavior

**Configuration**:
```yaml
rules:
- timeouts:
    # Match NGINX behavior exactly
    request: 600s
    backendRequest: 600s
  backendRefs:
  - name: legacy-service
    port: 80
```

### Scenario 5: Mixed Timeouts Per Path

**Requirements**:
- `/api/quick` → 10s timeout
- `/api/slow` → 300s timeout

**Configuration**:
```yaml
rules:
# Quick endpoints
- matches:
  - path:
      type: PathPrefix
      value: "/api/quick"
  timeouts:
    request: 10s
    backendRequest: 10s
  backendRefs:
  - name: api-service
    port: 8080

# Slow endpoints
- matches:
  - path:
      type: PathPrefix
      value: "/api/slow"
  timeouts:
    request: 300s
    backendRequest: 300s
  backendRefs:
  - name: api-service
    port: 8080
```

## Troubleshooting

### Problem: Getting 504 Gateway Timeout

**Symptoms**:
```
HTTP/1.1 504 Gateway Timeout
```

**Diagnosis**:
```bash
# Check HTTPRoute timeout configuration
kubectl get httproute my-route -o yaml | grep -A 5 timeouts

# Check Gateway default timeouts
kubectl get gateway my-gateway -o yaml

# Check backend response time
kubectl logs -n backend-namespace deploy/backend | grep duration
```

**Solutions**:

1. **Increase timeouts**:
```yaml
timeouts:
  request: 900s          # Increased from 600s
  backendRequest: 900s
```

2. **Check backend performance**:
```bash
# If backend is actually slow, fix the backend
# Don't just increase timeouts indefinitely
```

3. **Verify timeout constraint**:
```bash
# Ensure backendRequest ≤ request
ingress-to-gateway validate my-httproute.yaml
```

### Problem: Timeouts Too Long

**Symptoms**:
- Hung requests not timing out
- Resources exhausted
- Clients waiting too long

**Diagnosis**:
```bash
# Check current timeout configuration
kubectl get httproute -o yaml | grep -B 5 -A 5 timeouts

# Check actual request durations
# (use your monitoring tools)
```

**Solution**:
```yaml
# Reduce to appropriate values
timeouts:
  request: 30s           # Reduced from 600s
  backendRequest: 30s
```

### Problem: Constraint Violation

**Symptoms**:
```
Error: backendRequest timeout (600s) must be <= request timeout (300s)
```

**Solution**:
```yaml
# ❌ Invalid
timeouts:
  request: 300s
  backendRequest: 600s

# ✅ Fixed
timeouts:
  request: 600s          # Increased request
  backendRequest: 600s
```

### Problem: Different Behavior After Migration

**Symptoms**:
- Requests that worked before now timeout
- Or vice versa

**Diagnosis**:
```bash
# Check original NGINX timeouts
kubectl get ingress my-app -o yaml | grep timeout

# Compare with HTTPRoute timeouts
kubectl get httproute my-app -o yaml | grep -A 5 timeouts

# Check Gateway defaults
kubectl describe gateway my-gateway | grep -i timeout
```

**Solution**:
```yaml
# Match NGINX configuration exactly
# NGINX had: proxy-read-timeout: "600"
timeouts:
  request: 600s
  backendRequest: 600s
```

## Conversion Script Reference

When using `ingress-to-gateway convert`, timeouts are automatically converted:

```bash
# The tool automatically:
# 1. Detects proxy-read-timeout
# 2. Detects proxy-send-timeout
# 3. Uses maximum value
# 4. Sets BOTH request and backendRequest
# 5. Validates constraint

ingress-to-gateway convert my-ingress -n default

# Generated HTTPRoute will have:
# timeouts:
#   request: <value>s
#   backendRequest: <value>s
```

## Summary

✅ **Always set both timeout fields** (`request` and `backendRequest`)
✅ **Use same value** for both in most cases
✅ **Respect constraint**: `backendRequest ≤ request`
✅ **Match NGINX behavior**: Use same timeout values from NGINX config
✅ **Test after migration**: Verify timeout behavior
✅ **Monitor**: Track timeout errors (504s)
✅ **Document**: Comment why specific timeouts are set

## Next Steps

- Review [Annotation Mapping](ANNOTATION-MAPPING.md) for all annotation conversions
- Check [Troubleshooting](TROUBLESHOOTING.md) for common migration issues
- See [Migration Strategies](MIGRATION-STRATEGIES.md) for phased migration approaches
