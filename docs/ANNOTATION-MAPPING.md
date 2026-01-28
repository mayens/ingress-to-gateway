# Complete Annotation Mapping

Comprehensive guide for how NGINX Ingress annotations are mapped to Gateway API constructs.

## Table of Contents

- [Overview](#overview)
- [Routing & Rewrite](#routing--rewrite)
- [Redirects](#redirects)
- [Timeouts](#timeouts)
- [Backend Configuration](#backend-configuration)
- [TLS & Security](#tls--security)
- [Traffic Management](#traffic-management)
- [CORS](#cors)
- [Authentication](#authentication)
- [Rate Limiting](#rate-limiting)
- [Custom Configuration](#custom-configuration)
- [Unsupported Annotations](#unsupported-annotations)

## Overview

This document describes how each NGINX Ingress annotation is converted to Gateway API HTTPRoute configuration.

### Conversion Status Legend

- ✅ **Fully Supported**: Direct mapping available
- ⚠️ **Partially Supported**: Requires additional configuration
- ❌ **Not Supported**: No Gateway API equivalent (requires manual implementation)
- 🔍 **Manual Review**: Requires case-by-case evaluation

## Routing & Rewrite

### URL Rewrite

#### `nginx.ingress.kubernetes.io/rewrite-target`

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$2
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /api(/|$)(.*)
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 8080
```

**HTTPRoute Configuration:**
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: example-httproute
spec:
  parentRefs:
  - name: gateway-nginx
  hostnames:
  - "app.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: "/api/"
    filters:
    - type: URLRewrite
      urlRewrite:
        path:
          type: ReplaceFullPath
          replaceFullPath: "/$2"
    backendRefs:
    - name: api-service
      port: 8080
```

**Notes:**
- Capture groups from NGINX regex are mapped to Gateway API path modifiers
- Complex regex patterns may require manual adjustment

#### `nginx.ingress.kubernetes.io/use-regex`

**Status**: ⚠️ Partially Supported

Gateway API uses PathPrefix or Exact matching, not full regex. Complex patterns need to be broken down.

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/use-regex: "true"
spec:
  rules:
  - http:
      paths:
      - path: /api/(v[0-9]+)/.*
        pathType: Prefix
```

**HTTPRoute Configuration:**
```yaml
# Must be converted to multiple explicit path matches
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/api/v1/"
- matches:
  - path:
      type: PathPrefix
      value: "/api/v2/"
```

### App Root Redirect

#### `nginx.ingress.kubernetes.io/app-root`

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/app-root: /app
spec:
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

**HTTPRoute Configuration:**
```yaml
rules:
- matches:
  - path:
      type: Exact
      value: "/"
  filters:
  - type: RequestRedirect
    requestRedirect:
      path:
        type: ReplaceFullPath
        replaceFullPath: "/app"
      statusCode: 302
  backendRefs: []
- matches:
  - path:
      type: PathPrefix
      value: "/"
  backendRefs:
  - name: app-service
    port: 80
```

## Redirects

### SSL Redirect

#### `nginx.ingress.kubernetes.io/ssl-redirect`

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
```

**HTTPRoute Configuration:**
```yaml
# Create two HTTPRoutes: one for HTTP redirect, one for HTTPS traffic
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: example-http-redirect
spec:
  parentRefs:
  - name: gateway-nginx
    sectionName: http
  rules:
  - filters:
    - type: RequestRedirect
      requestRedirect:
        scheme: https
        statusCode: 301
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: example-https
spec:
  parentRefs:
  - name: gateway-nginx
    sectionName: https
  rules:
  - backendRefs:
    - name: app-service
      port: 80
```

#### `nginx.ingress.kubernetes.io/force-ssl-redirect`

**Status**: ✅ Fully Supported

Same as `ssl-redirect` but always enforced regardless of TLS configuration.

#### `nginx.ingress.kubernetes.io/permanent-redirect`

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/permanent-redirect: https://new-site.example.com
```

**HTTPRoute Configuration:**
```yaml
rules:
- filters:
  - type: RequestRedirect
    requestRedirect:
      hostname: new-site.example.com
      statusCode: 301
```

#### `nginx.ingress.kubernetes.io/temporal-redirect`

**Status**: ✅ Fully Supported

**HTTPRoute Configuration:**
```yaml
rules:
- filters:
  - type: RequestRedirect
    requestRedirect:
      hostname: temp-site.example.com
      statusCode: 302
```

## Timeouts

### Proxy Timeouts

#### `nginx.ingress.kubernetes.io/proxy-read-timeout`

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
```

**HTTPRoute Configuration:**
```yaml
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
```

**Notes:**
- **Both timeout fields are set** for complete timeout control
- `backendRequest`: Timeout for gateway → backend communication
- `request`: Total timeout for client → gateway → backend → client
- Constraint: `backendRequest ≤ request`

See [Timeout Configuration](TIMEOUT-CONFIGURATION.md) for detailed explanation.

#### `nginx.ingress.kubernetes.io/proxy-send-timeout`

**Status**: ✅ Fully Supported

Maps to same timeout fields as `proxy-read-timeout`.

#### `nginx.ingress.kubernetes.io/proxy-connect-timeout`

**Status**: ⚠️ Partially Supported

Gateway API doesn't have a separate connect timeout. Maps to `backendRequest` timeout.

### Client Timeouts

#### `nginx.ingress.kubernetes.io/client-body-timeout`

**Status**: ❌ Not Supported

No direct Gateway API equivalent. Must be configured at the Gateway level.

## Backend Configuration

### Proxy Body Size

#### `nginx.ingress.kubernetes.io/proxy-body-size`

**Status**: ⚠️ Partially Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
```

**HTTPRoute Configuration:**

No direct mapping. Must be configured using Gateway-specific configuration:

```yaml
# Example for NGINX Gateway Fabric
apiVersion: gateway.nginx.org/v1alpha1
kind: ClientSettingsPolicy
metadata:
  name: client-settings
spec:
  targetRef:
    kind: HTTPRoute
    name: example-httproute
  body:
    maxSize: 100m
```

### Backend Protocol

#### `nginx.ingress.kubernetes.io/backend-protocol`

**Status**: ⚠️ Partially Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
```

**HTTPRoute Configuration:**

Gateway API uses Service AppProtocol:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  ports:
  - port: 443
    protocol: TCP
    appProtocol: https  # Indicates backend uses HTTPS
```

Or use BackendTLSPolicy (Gateway API v1.1+):

```yaml
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: BackendTLSPolicy
metadata:
  name: backend-tls
spec:
  targetRef:
    kind: Service
    name: backend-service
  tls:
    mode: Terminate
    certificateAuthorityRefs:
    - name: backend-ca
```

### Upstream Hash By

#### `nginx.ingress.kubernetes.io/upstream-hash-by`

**Status**: ❌ Not Supported

Session affinity must be configured differently. Use Service sessionAffinity:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: app-service
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800
```

## TLS & Security

### TLS Configuration

#### TLS Termination

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
spec:
  tls:
  - hosts:
    - app.example.com
    - api.example.com
    secretName: example-tls
  rules:
  - host: app.example.com
    # ...
```

**Gateway Configuration:**
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-nginx
spec:
  gatewayClassName: nginx
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: "*.example.com"
    tls:
      mode: Terminate
      certificateRefs:
      - name: example-tls
```

**HTTPRoute Configuration:**
```yaml
spec:
  parentRefs:
  - name: gateway-nginx
    sectionName: https  # References HTTPS listener
  hostnames:
  - "app.example.com"
  - "api.example.com"
```

#### `nginx.ingress.kubernetes.io/auth-tls-secret`

**Status**: ⚠️ Partially Supported

mTLS configuration moves to Gateway:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
spec:
  listeners:
  - name: https-mtls
    protocol: HTTPS
    port: 443
    tls:
      mode: Terminate
      certificateRefs:
      - name: server-cert
      options:
        gateway.nginx.org/client-ca-secret: client-ca-secret
```

## Traffic Management

### Canary Deployments

#### `nginx.ingress.kubernetes.io/canary`

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
# Main Ingress
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-main
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        backend:
          service:
            name: app-v1
            port:
              number: 80
---
# Canary Ingress
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-canary
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "20"
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        backend:
          service:
            name: app-v2
            port:
              number: 80
```

**HTTPRoute Configuration:**
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: app-httproute
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
    backendRefs:
    - name: app-v1
      port: 80
      weight: 80
    - name: app-v2
      port: 80
      weight: 20
```

#### `nginx.ingress.kubernetes.io/canary-weight`

**Status**: ✅ Fully Supported

Maps directly to `backendRefs[].weight`.

#### `nginx.ingress.kubernetes.io/canary-by-header`

**Status**: ✅ Fully Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-by-header: "X-Canary"
    nginx.ingress.kubernetes.io/canary-by-header-value: "always"
```

**HTTPRoute Configuration:**
```yaml
rules:
# Canary traffic (with header)
- matches:
  - path:
      type: PathPrefix
      value: "/"
    headers:
    - name: X-Canary
      value: always
  backendRefs:
  - name: app-v2
    port: 80
# Normal traffic (without header)
- matches:
  - path:
      type: PathPrefix
      value: "/"
  backendRefs:
  - name: app-v1
    port: 80
```

#### `nginx.ingress.kubernetes.io/canary-by-header-pattern`

**Status**: ✅ Fully Supported

**HTTPRoute Configuration:**
```yaml
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/"
    headers:
    - name: X-Canary
      type: RegularExpression
      value: "^(alpha|beta)$"
  backendRefs:
  - name: app-v2
    port: 80
```

### Traffic Mirroring

#### `nginx.ingress.kubernetes.io/mirror-uri`

**Status**: ⚠️ Partially Supported

**HTTPRoute Configuration:**
```yaml
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/"
  filters:
  - type: RequestMirror
    requestMirror:
      backendRef:
        name: mirror-service
        port: 80
  backendRefs:
  - name: main-service
    port: 80
```

**Notes:**
- Gateway API RequestMirror is simpler than NGINX mirror
- Percentage-based mirroring not supported

## CORS

### CORS Configuration

#### `nginx.ingress.kubernetes.io/enable-cors`

**Status**: ⚠️ Partially Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/enable-cors: "true"
    nginx.ingress.kubernetes.io/cors-allow-origin: "https://example.com"
    nginx.ingress.kubernetes.io/cors-allow-methods: "GET, POST, OPTIONS"
    nginx.ingress.kubernetes.io/cors-allow-headers: "Authorization, Content-Type"
    nginx.ingress.kubernetes.io/cors-allow-credentials: "true"
    nginx.ingress.kubernetes.io/cors-max-age: "86400"
```

**HTTPRoute Configuration:**

CORS is not standardized in Gateway API v1.0. Requires gateway-specific policy:

```yaml
# Example for NGINX Gateway Fabric
apiVersion: gateway.nginx.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: cors-policy
spec:
  targetRef:
    kind: HTTPRoute
    name: example-httproute
  cors:
    allowOrigins:
    - "https://example.com"
    allowMethods:
    - GET
    - POST
    - OPTIONS
    allowHeaders:
    - Authorization
    - Content-Type
    allowCredentials: true
    maxAge: 86400s
```

## Authentication

### Basic Auth

#### `nginx.ingress.kubernetes.io/auth-type`

**Status**: ⚠️ Partially Supported

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/auth-type: basic
    nginx.ingress.kubernetes.io/auth-secret: basic-auth
    nginx.ingress.kubernetes.io/auth-realm: "Authentication Required"
```

**HTTPRoute Configuration:**

Authentication is not standardized in Gateway API v1.0. Requires gateway-specific policy:

```yaml
# Example for NGINX Gateway Fabric
apiVersion: gateway.nginx.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: auth-policy
spec:
  targetRef:
    kind: HTTPRoute
    name: example-httproute
  basicAuth:
    secretRef:
      name: basic-auth
    realm: "Authentication Required"
```

### External Auth

#### `nginx.ingress.kubernetes.io/auth-url`

**Status**: ❌ Not Supported

External authentication requires custom implementation or gateway-specific extension.

## Rate Limiting

#### `nginx.ingress.kubernetes.io/limit-rps`

**Status**: ❌ Not Supported

Rate limiting is not standardized in Gateway API v1.0.

**Gateway-Specific Policy Example:**
```yaml
apiVersion: gateway.nginx.org/v1alpha1
kind: RateLimitPolicy
metadata:
  name: rate-limit
spec:
  targetRef:
    kind: HTTPRoute
    name: example-httproute
  rateLimit:
    requests: 100
    window: 1m
    burst: 50
```

## Custom Configuration

### Configuration Snippets

#### `nginx.ingress.kubernetes.io/configuration-snippet`

**Status**: 🔍 Manual Review Required

**Ingress Configuration:**
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/configuration-snippet: |
      more_set_headers "X-Custom-Header: value";
      if ($request_uri ~* /admin) {
        return 403;
      }
```

**Migration Strategy:**

1. **Analyze the snippet** to understand what it does
2. **Find Gateway API equivalent** if possible
3. **Use gateway-specific policies** if needed
4. **Implement at application level** as last resort

**Example - Custom Headers:**
```yaml
# Can be done with RequestHeaderModifier
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/"
  filters:
  - type: RequestHeaderModifier
    requestHeaderModifier:
      add:
      - name: X-Custom-Header
        value: value
```

**Example - Path-based access control:**
```yaml
# Create separate HTTPRoute with no backend
rules:
- matches:
  - path:
      type: PathPrefix
      value: "/admin"
  # No backendRefs = 404 response
  backendRefs: []
```

#### `nginx.ingress.kubernetes.io/server-snippet`

**Status**: 🔍 Manual Review Required

Server-level configuration must be moved to Gateway configuration.

## Unsupported Annotations

The following annotations have no direct Gateway API equivalent:

- `nginx.ingress.kubernetes.io/affinity` - Use Service sessionAffinity
- `nginx.ingress.kubernetes.io/affinity-mode` - Use Service sessionAffinity
- `nginx.ingress.kubernetes.io/service-upstream` - Configure at Gateway level
- `nginx.ingress.kubernetes.io/upstream-vhost` - Configure backend Service
- `nginx.ingress.kubernetes.io/whitelist-source-range` - Use gateway policy
- `nginx.ingress.kubernetes.io/proxy-buffering` - Configure at Gateway level
- `nginx.ingress.kubernetes.io/proxy-buffer-size` - Configure at Gateway level
- Custom Lua snippets - Requires gateway extension or sidecar

## Migration Strategy for Unsupported Features

1. **Check Gateway Implementation**: Different Gateway implementations (NGINX, Envoy, Traefik) may support features via custom policies

2. **Move to Application**: Some features can be implemented at the application level

3. **Use Service Mesh**: For advanced traffic management, consider a service mesh

4. **Gateway Extensions**: Use gateway-specific CRDs for advanced features

5. **Wait for Gateway API**: Some features are planned for future Gateway API versions

## Next Steps

- See [Timeout Configuration](TIMEOUT-CONFIGURATION.md) for detailed timeout mappings
- Review [Migration Strategies](MIGRATION-STRATEGIES.md) for phased migration approaches
- Check [Troubleshooting](TROUBLESHOOTING.md) for common issues

## Summary Table

| NGINX Annotation | Gateway API | Status |
|-----------------|-------------|--------|
| `rewrite-target` | URLRewrite filter | ✅ Full |
| `app-root` | RequestRedirect filter | ✅ Full |
| `ssl-redirect` | RequestRedirect filter | ✅ Full |
| `permanent-redirect` | RequestRedirect filter | ✅ Full |
| `proxy-read-timeout` | timeouts.backendRequest | ✅ Full |
| `proxy-send-timeout` | timeouts.backendRequest | ✅ Full |
| `backend-protocol` | Service appProtocol | ⚠️ Partial |
| `canary` | backendRefs weights | ✅ Full |
| `canary-weight` | backendRefs.weight | ✅ Full |
| `canary-by-header` | header matches | ✅ Full |
| `enable-cors` | Gateway policy | ⚠️ Gateway-specific |
| `auth-type` | Gateway policy | ⚠️ Gateway-specific |
| `configuration-snippet` | Manual review | 🔍 Case-by-case |
| `limit-rps` | Gateway policy | ❌ Not supported |
| `whitelist-source-range` | Gateway policy | ❌ Not supported |
