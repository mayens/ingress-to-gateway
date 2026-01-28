# Migration Strategies

Proven strategies for migrating from Ingress-NGINX to Gateway API with minimal risk and downtime.

## Table of Contents

- [Strategy Overview](#strategy-overview)
- [Strategy 1: Big Bang Migration](#strategy-1-big-bang-migration)
- [Strategy 2: Blue-Green Migration](#strategy-2-blue-green-migration)
- [Strategy 3: Canary Migration](#strategy-3-canary-migration)
- [Strategy 4: Phased Migration](#strategy-4-phased-migration)
- [Strategy 5: Shadow Traffic Migration](#strategy-5-shadow-traffic-migration)
- [Choosing the Right Strategy](#choosing-the-right-strategy)
- [Risk Mitigation](#risk-mitigation)

## Strategy Overview

| Strategy | Downtime | Complexity | Risk | Best For |
|----------|----------|------------|------|----------|
| Big Bang | Minimal | Low | Medium | Small deployments, dev/test |
| Blue-Green | None | Medium | Low | Production with quick rollback needs |
| Canary | None | High | Very Low | Large production deployments |
| Phased | None | Medium | Low | Multiple applications |
| Shadow | None | High | Very Low | Critical production systems |

## Strategy 1: Big Bang Migration

**Overview**: Convert all Ingress resources at once during a maintenance window.

### When to Use

- ✅ Development/staging environments
- ✅ Small number of Ingress resources (<10)
- ✅ Non-critical applications
- ✅ Weekend or off-peak deployments
- ✅ Well-tested configurations

### When NOT to Use

- ❌ Production systems with strict SLA
- ❌ Large number of complex Ingress resources
- ❌ 24/7 critical applications
- ❌ Untested Gateway API setup

### Implementation Steps

#### 1. Preparation

```bash
# Audit all resources
ingress-to-gateway audit --all-namespaces --detailed > audit-report.txt

# Review audit report
cat audit-report.txt

# Backup existing Ingress
kubectl get ingress --all-namespaces -o yaml > ingress-backup-$(date +%Y%m%d).yaml
```

#### 2. Convert All Resources

```bash
# Batch convert
ingress-to-gateway batch --all-namespaces -o httproutes/

# Validate all
for file in httproutes/**/*.yaml; do
  ingress-to-gateway validate "$file" || exit 1
done
```

#### 3. Schedule Maintenance Window

```bash
# Create maintenance notice
cat > maintenance-notice.txt <<EOF
MAINTENANCE WINDOW
Date: $(date)
Duration: 2 hours
Scope: Gateway API migration
Expected Impact: Brief service interruption during cutover
EOF
```

#### 4. Execute Migration

```bash
#!/bin/bash
# migration-script.sh

# Stop accepting new deployments
kubectl scale deployment ingress-controller -n ingress-nginx --replicas=0

# Apply Gateway resources
kubectl apply -f gateway.yaml

# Apply all HTTPRoutes
kubectl apply -f httproutes/ --recursive

# Wait for HTTPRoutes to be ready
kubectl wait --for=condition=Accepted httproute --all --timeout=5m

# Verify traffic
./verify-traffic.sh

# If successful, delete old Ingress
if [ $? -eq 0 ]; then
  kubectl delete ingress --all --all-namespaces
  echo "Migration successful"
else
  echo "Migration failed, rolling back"
  kubectl apply -f ingress-backup-$(date +%Y%m%d).yaml
  kubectl scale deployment ingress-controller -n ingress-nginx --replicas=3
  exit 1
fi
```

### Pros

✅ Simple and straightforward
✅ Fast completion
✅ Easy to understand
✅ Single point of testing

### Cons

❌ Requires maintenance window
❌ All-or-nothing approach
❌ Higher risk if issues occur
❌ Pressure to rollback quickly

## Strategy 2: Blue-Green Migration

**Overview**: Run both Ingress and Gateway in parallel, switch DNS/load balancer at once.

### When to Use

- ✅ Production systems
- ✅ Need for quick rollback
- ✅ External traffic (DNS-based)
- ✅ Sufficient infrastructure capacity

### Architecture

```
                    ┌──────────────┐
                    │  DNS / LB    │
                    └──────┬───────┘
                           │
              ┌────────────┴────────────┐
              │ Switch traffic here     │
              │                         │
       ┌──────▼───────┐         ┌─────▼─────────┐
       │  BLUE (Old)  │         │  GREEN (New)  │
       │              │         │               │
       │   Ingress    │         │   Gateway     │
       │  Controller  │         │  Controller   │
       └──────┬───────┘         └─────┬─────────┘
              │                       │
       ┌──────▼───────┐         ┌─────▼─────────┐
       │   Ingress    │         │   HTTPRoute   │
       │  Resources   │         │   Resources   │
       └──────────────┘         └───────────────┘
```

### Implementation Steps

#### 1. Deploy Green Environment

```bash
# Install Gateway API controller (without traffic)
kubectl apply -f gateway-controller.yaml

# Create Gateway resource
kubectl apply -f gateway.yaml

# Convert and apply HTTPRoutes
ingress-to-gateway batch --all-namespaces -o httproutes/
kubectl apply -f httproutes/ --recursive

# Verify HTTPRoutes are accepted
kubectl get httproute --all-namespaces
```

#### 2. Test Green Environment

```bash
# Get Gateway external IP
GATEWAY_IP=$(kubectl get gateway gateway-nginx -o jsonpath='{.status.addresses[0].value}')

# Test all endpoints (modify /etc/hosts or use curl -H Host)
for host in app.example.com api.example.com; do
  echo "Testing $host..."
  curl -H "Host: $host" http://$GATEWAY_IP/ -I
done

# Run automated tests
./run-integration-tests.sh $GATEWAY_IP

# Load test
hey -z 30s -c 10 -H "Host: app.example.com" http://$GATEWAY_IP/
```

#### 3. Switch Traffic

```bash
# Option A: Update DNS
# Change DNS A record from Blue IP to Green IP
# TTL should be low (60s) for quick switching

# Option B: Update LoadBalancer
kubectl patch service gateway-lb -p '{"spec":{"externalIPs":["NEW_IP"]}}'

# Option C: Update Ingress external IP
# If using cloud provider, update forwarding rules
```

#### 4. Monitor

```bash
# Monitor both environments
watch kubectl get httproute --all-namespaces
watch kubectl get ingress --all-namespaces

# Monitor Gateway metrics
kubectl logs -n gateway-system -l app=gateway -f

# Monitor application metrics
# (Use your monitoring tools)
```

#### 5. Rollback (if needed)

```bash
# Simply switch DNS/LB back to Blue
# No changes to cluster needed

# Update DNS back to Blue IP
# Or
kubectl patch service gateway-lb -p '{"spec":{"externalIPs":["OLD_IP"]}}'
```

#### 6. Cleanup (after success)

```bash
# Wait 24-48 hours of stable operation
# Then remove Blue environment
kubectl delete ingress --all --all-namespaces
kubectl delete -f ingress-controller.yaml
```

### Pros

✅ Zero downtime
✅ Instant rollback
✅ Both environments testable
✅ Low risk

### Cons

❌ Double infrastructure cost during migration
❌ DNS propagation delay (if using DNS)
❌ More complex setup
❌ Need external traffic routing control

## Strategy 3: Canary Migration

**Overview**: Gradually shift traffic percentage from Ingress to Gateway.

### When to Use

- ✅ Large production deployments
- ✅ Risk-averse organizations
- ✅ Need gradual validation
- ✅ Complex applications

### Architecture

```
                    ┌──────────────┐
                    │  Load Balancer │
                    │  (Traffic Split)│
                    └──────┬───────┘
                           │
              ┌────────────┴────────────┐
              │                         │
       ┌──────▼───────┐         ┌─────▼─────────┐
       │   95% →      │         │   5% →        │
       │   Ingress    │         │   Gateway     │
       │  Controller  │         │  Controller   │
       └──────────────┘         └───────────────┘
              ↓                         ↓
       [ Gradually decrease ]   [ Gradually increase ]
              ↓                         ↓
           0% →                     100% →
```

### Implementation Steps

#### 1. Initial Setup (0% to Gateway)

```bash
# Deploy Gateway alongside Ingress
kubectl apply -f gateway-controller.yaml
kubectl apply -f gateway.yaml

# Convert HTTPRoutes
ingress-to-gateway batch --all-namespaces -o httproutes/
kubectl apply -f httproutes/ --recursive
```

#### 2. Configure Traffic Split

**Option A: Using LoadBalancer weights**

```bash
# Configure LB to split traffic
# Example for AWS ALB
aws elbv2 modify-listener \
  --listener-arn $LISTENER_ARN \
  --default-actions \
  Type=forward,TargetGroupArn=$INGRESS_TG,Weight=95 \
  Type=forward,TargetGroupArn=$GATEWAY_TG,Weight=5
```

**Option B: Using Service Mesh (Istio)**

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: traffic-split
spec:
  hosts:
  - "*.example.com"
  http:
  - match:
    - uri:
        prefix: "/"
    route:
    - destination:
        host: ingress-controller
      weight: 95
    - destination:
        host: gateway-controller
      weight: 5
```

**Option C: Using Flagger (GitOps)**

```yaml
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: gateway-migration
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gateway-controller
  service:
    port: 80
  analysis:
    interval: 1h
    threshold: 5
    maxWeight: 100
    stepWeight: 10
    metrics:
    - name: request-success-rate
      thresholdRange:
        min: 99
    - name: request-duration
      thresholdRange:
        max: 500
```

#### 3. Gradual Rollout Schedule

```bash
# Week 1: 5% to Gateway
# - Monitor errors
# - Validate functionality
# - Check performance

# Week 2: 20% to Gateway
# - Continue monitoring
# - Validate under increased load

# Week 3: 50% to Gateway
# - Peak traffic testing
# - Performance validation

# Week 4: 100% to Gateway
# - Full migration
# - Keep Ingress as backup for 1 week
```

#### 4. Monitoring Script

```bash
#!/bin/bash
# monitor-canary.sh

while true; do
  # Check error rates
  INGRESS_ERRORS=$(kubectl logs -n ingress-nginx --tail=100 | grep -c "error")
  GATEWAY_ERRORS=$(kubectl logs -n gateway-system --tail=100 | grep -c "error")

  echo "Ingress errors: $INGRESS_ERRORS"
  echo "Gateway errors: $GATEWAY_ERRORS"

  # Check HTTPRoute status
  kubectl get httproute --all-namespaces -o jsonpath='{range .items[*]}{.metadata.name}: {.status.parents[*].conditions[?(@.type=="Accepted")].status}{"\n"}{end}'

  sleep 60
done
```

### Pros

✅ Lowest risk
✅ Gradual validation
✅ Easy rollback at any stage
✅ Continuous monitoring

### Cons

❌ Longest migration timeline
❌ Requires traffic split capability
❌ Complex monitoring
❌ Higher operational overhead

## Strategy 4: Phased Migration

**Overview**: Migrate one application/namespace at a time.

### When to Use

- ✅ Multiple independent applications
- ✅ Different teams owning different services
- ✅ Want to learn from early migrations
- ✅ Diverse application requirements

### Implementation Steps

#### 1. Prioritize Applications

```bash
# Audit all applications
ingress-to-gateway audit --all-namespaces --detailed -o json > audit.json

# Rank by complexity (migrate simple ones first)
cat audit.json | jq -r '.[] | "\(.ComplexityScore) \(.Namespace)/\(.Name)"' | sort -n

# Create migration order
cat > migration-order.txt <<EOF
Phase 1 (Week 1): Low complexity
- default/simple-app (complexity: 5)
- staging/test-app (complexity: 6)

Phase 2 (Week 2): Medium complexity
- production/api-gateway (complexity: 15)
- production/web-app (complexity: 18)

Phase 3 (Week 3): High complexity
- production/legacy-app (complexity: 25)
- production/complex-app (complexity: 30)
EOF
```

#### 2. Migrate First Application

```bash
# Phase 1: simple-app
ingress-to-gateway convert simple-app -n default -o simple-app-httproute.yaml
ingress-to-gateway validate simple-app-httproute.yaml
kubectl apply -f simple-app-httproute.yaml

# Test thoroughly
./test-application.sh simple-app default

# Keep Ingress as backup for 1 week
# Delete after validation
kubectl delete ingress simple-app -n default
```

#### 3. Learn and Iterate

```bash
# Document lessons learned
cat >> migration-notes.md <<EOF
## Phase 1 Lessons

### simple-app migration
- Issue: Timeout values too low
- Solution: Increased to 600s
- Time taken: 2 hours
- Downtime: 0 minutes

### test-app migration
- Issue: Missing Gateway listener
- Solution: Added HTTPS listener
- Time taken: 3 hours
- Downtime: 0 minutes
EOF
```

#### 4. Continue with Remaining Phases

Repeat for each application, applying lessons learned.

### Pros

✅ Learn from each migration
✅ Minimal blast radius
✅ Independent rollback per app
✅ Team can specialize

### Cons

❌ Long overall timeline
❌ Maintaining two systems longer
❌ Potential configuration drift
❌ More overhead per migration

## Strategy 5: Shadow Traffic Migration

**Overview**: Route real traffic to Gateway in read-only mode for validation.

### When to Use

- ✅ Mission-critical applications
- ✅ Zero tolerance for errors
- ✅ Need extensive validation
- ✅ Complex traffic patterns

### Implementation Steps

#### 1. Setup Traffic Mirroring

```bash
# Configure Ingress to mirror traffic to Gateway
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-with-mirror
  annotations:
    nginx.ingress.kubernetes.io/mirror-target: http://gateway-nginx.gateway-system.svc.cluster.local
    nginx.ingress.kubernetes.io/mirror-request-body: "on"
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
EOF
```

#### 2. Deploy Gateway (Shadow Mode)

```bash
# Deploy Gateway and HTTPRoutes
kubectl apply -f gateway.yaml
ingress-to-gateway convert app-with-mirror -n default -o app-httproute.yaml
kubectl apply -f app-httproute.yaml

# Gateway receives copied traffic but responses are discarded
```

#### 3. Compare Results

```bash
#!/bin/bash
# compare-responses.sh

# Log Ingress responses
kubectl logs -n ingress-nginx deploy/ingress-controller | grep "app.example.com" > ingress-logs.txt

# Log Gateway responses
kubectl logs -n gateway-system -l app=gateway | grep "app.example.com" > gateway-logs.txt

# Compare error rates
INGRESS_ERRORS=$(grep -c "5[0-9][0-9]" ingress-logs.txt)
GATEWAY_ERRORS=$(grep -c "5[0-9][0-9]" gateway-logs.txt)

echo "Ingress errors: $INGRESS_ERRORS"
echo "Gateway errors: $GATEWAY_ERRORS"

# Compare latencies
INGRESS_P99=$(grep "request_time" ingress-logs.txt | awk '{print $NF}' | sort -n | tail -1)
GATEWAY_P99=$(grep "duration" gateway-logs.txt | awk '{print $NF}' | sort -n | tail -1)

echo "Ingress P99: $INGRESS_P99"
echo "Gateway P99: $GATEWAY_P99"
```

#### 4. Validate for Extended Period

```bash
# Run shadow mode for 1-2 weeks
# Monitor continuously
./compare-responses.sh >> comparison-$(date +%Y%m%d).log

# Check for discrepancies
if [ $GATEWAY_ERRORS -le $INGRESS_ERRORS ]; then
  echo "Gateway performing equal or better"
else
  echo "Gateway has more errors, investigate"
fi
```

#### 5. Cutover When Ready

Once validated, proceed with one of the other strategies (Blue-Green or Canary).

### Pros

✅ Lowest risk possible
✅ Real traffic validation
✅ No impact on users
✅ Extended validation period

### Cons

❌ Longest overall timeline
❌ Double resource consumption
❌ Complex comparison logic
❌ Requires mirroring capability

## Choosing the Right Strategy

### Decision Tree

```
Are you in production?
├─ No → Big Bang
└─ Yes
   └─ Can you afford downtime?
      ├─ Yes (< 5 min) → Big Bang
      └─ No
         └─ Do you have traffic split capability?
            ├─ No → Phased
            └─ Yes
               └─ How risk-averse?
                  ├─ Low → Blue-Green
                  ├─ Medium → Canary
                  └─ Very High → Shadow + Canary
```

### By Organization Size

- **Startup/Small Team**: Big Bang or Phased
- **Medium Company**: Blue-Green or Phased
- **Enterprise**: Canary or Shadow + Canary

### By Application Criticality

- **Dev/Test**: Big Bang
- **Staging**: Blue-Green
- **Production (non-critical)**: Blue-Green or Phased
- **Production (critical)**: Canary or Shadow + Canary

## Risk Mitigation

### Pre-Migration Checklist

```bash
# ✅ Gateway API CRDs installed
kubectl get crd gateways.gateway.networking.k8s.io

# ✅ Gateway controller installed
kubectl get deployment -n gateway-system

# ✅ Gateway resource created
kubectl get gateway

# ✅ Audit completed
ingress-to-gateway audit --all-namespaces --detailed

# ✅ Backups taken
kubectl get ingress --all-namespaces -o yaml > backup.yaml

# ✅ HTTPRoutes validated
ingress-to-gateway validate *.yaml

# ✅ Test environment validated
./test-migration.sh

# ✅ Rollback plan documented
cat ROLLBACK.md

# ✅ Monitoring configured
kubectl get servicemonitor

# ✅ Team trained
# (Training sessions completed)

# ✅ Stakeholders notified
# (Communication sent)
```

### Rollback Plans

**For Big Bang:**
```bash
kubectl apply -f ingress-backup-$(date +%Y%m%d).yaml
kubectl delete httproute --all --all-namespaces
kubectl scale deployment ingress-controller -n ingress-nginx --replicas=3
```

**For Blue-Green:**
```bash
# Simply switch traffic back
# Update DNS or LoadBalancer
```

**For Canary:**
```bash
# Decrease Gateway weight to 0%
aws elbv2 modify-listener --listener-arn $LISTENER_ARN --default-actions Type=forward,TargetGroupArn=$INGRESS_TG,Weight=100
```

**For Phased:**
```bash
# Rollback specific application
kubectl apply -f backup/app-ingress.yaml
kubectl delete httproute app-httproute -n namespace
```

### Monitoring During Migration

```bash
# Key metrics to watch
watch kubectl get httproute --all-namespaces
watch kubectl get gateway

# Application metrics
# - Request rate
# - Error rate (4xx, 5xx)
# - Latency (P50, P95, P99)
# - Throughput

# Infrastructure metrics
# - Gateway CPU/Memory
# - Network throughput
# - Connection count

# Business metrics
# - User sessions
# - Transaction success rate
# - Revenue impact (if applicable)
```

## Next Steps

- See [Timeout Configuration](TIMEOUT-CONFIGURATION.md) for timeout best practices
- Review [Troubleshooting](TROUBLESHOOTING.md) for common issues
- Check [Annotation Mapping](ANNOTATION-MAPPING.md) for feature equivalents
