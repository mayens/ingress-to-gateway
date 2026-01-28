# Interactive Mode Guide

Step-by-step guided migration with the interactive wizard.

## Table of Contents

- [Overview](#overview)
- [When to Use](#when-to-use)
- [Starting the Wizard](#starting-the-wizard)
- [Wizard Steps](#wizard-steps)
- [Example Session](#example-session)
- [Tips and Best Practices](#tips-and-best-practices)

## Overview

Interactive mode provides a **guided, step-by-step experience** for migrating Ingress resources to Gateway API HTTPRoute. It's perfect for users who prefer a visual, menu-driven interface over command-line flags.

### What the Wizard Does

1. ✅ Lists available namespaces and Ingress resources
2. ✅ Analyzes migration complexity and readiness
3. ✅ Guides through configuration options
4. ✅ Previews generated HTTPRoute before saving
5. ✅ Validates output for correctness
6. ✅ Saves or displays the result

## When to Use

### Perfect For:
- 👤 **First-time users** learning the migration process
- 🎓 **Learning mode** to understand conversion options
- 🔍 **Exploratory analysis** of different configurations
- 🧩 **Complex migrations** requiring careful review
- 📋 **Guided workflows** instead of memorizing flags

### Not Ideal For:
- 🤖 **Automation** and CI/CD pipelines (use `convert` command)
- 📦 **Batch operations** on many Ingress (use `batch` command)
- ⚡ **Quick conversions** when you know exact flags

## Starting the Wizard

### Basic Usage

```bash
# Start interactive mode
ingress-to-gateway interactive

# With specific kubeconfig
ingress-to-gateway interactive --kubeconfig ~/.kube/prod-config

# With specific context
ingress-to-gateway interactive --context production
```

### Requirements

- ✅ Access to Kubernetes cluster (via kubeconfig)
- ✅ Permissions to list Ingress resources
- ✅ Terminal with interactive input support

## Wizard Steps

### Step 1: Select Namespace

```
─────────────────────────────────────────────────────────────
Step 1: Select Namespace
─────────────────────────────────────────────────────────────

Current namespace: default

Options:
  1. Use current namespace
  2. List all namespaces
  3. Enter namespace manually

Select option [1-3]: _
```

**What to do:**
- Choose **1** to use your current namespace
- Choose **2** to see all available namespaces
- Choose **3** to type a namespace name manually

**Tip**: Most users choose option 1 or 2.

### Step 2: Select Ingress Resource

```
─────────────────────────────────────────────────────────────
Step 2: Select Ingress Resource
─────────────────────────────────────────────────────────────

Found 3 Ingress resource(s) in namespace 'default':

  1. my-app-ingress
     Class: nginx
     Hosts: 2 | Rules: 3 | TLS: true

  2. api-ingress
     Class: nginx
     Hosts: 1 | Rules: 2 | TLS: false

  3. legacy-ingress
     Class: nginx
     Hosts: 5 | Rules: 10 | TLS: true

Select Ingress [1-3]: _
```

**What to do:**
- Review the listed Ingress resources
- Note the number of hosts, rules, and TLS configuration
- Select the Ingress you want to migrate

**Tip**: Start with simpler Ingress (fewer hosts/rules) for your first migration.

### Step 3: Migration Analysis

```
─────────────────────────────────────────────────────────────
Step 3: Migration Analysis
─────────────────────────────────────────────────────────────

Ingress: default/my-app-ingress
Ingress Class: nginx
Hosts: 2 | Paths: 3 | TLS: true

✓ Detected Features:
  • URL_REWRITE
  • TLS_TERMINATION
  • PROXY_READ_TIMEOUT

✅ Migration Readiness: READY (Complexity: 8)

💡 Recommendations:
  • Use 'single' split mode for optimal Gateway API resource usage
  • Both timeouts.request and timeouts.backendRequest will be set
  • Ensure Gateway has matching HTTPS listeners configured

Press Enter to continue...
```

**What the analysis shows:**
- **Features detected**: NGINX annotations found in your Ingress
- **Readiness level**:
  - ✅ **READY**: Simple migration, no issues
  - ⚠️ **MOSTLY_READY**: Mostly straightforward, minor issues
  - ⚠️ **COMPLEX**: Multiple complex features
  - ❌ **MANUAL_REVIEW_REQUIRED**: Custom snippets or unsupported features
- **Complexity score**: Numeric score (higher = more complex)
- **Recommendations**: Suggested best practices for this Ingress

**Tip**: If status is MANUAL_REVIEW_REQUIRED, review issues carefully before proceeding.

### Step 4: Configure Conversion Options

```
─────────────────────────────────────────────────────────────
Step 4: Configure Conversion Options
─────────────────────────────────────────────────────────────

Select split mode:
  1. Single   - One HTTPRoute for all hostnames (recommended)
  2. Per-host - Separate HTTPRoute per hostname
  3. Per-pattern - Group by hostname patterns

Split mode [1-3]: 1
✓ Split mode: single

Gateway name (default: gateway-nginx):
Gateway name (Enter for default): _
✓ Gateway: gateway-nginx

Gateway class (default: nginx):
Gateway class (Enter for default): _
✓ Gateway class: nginx
```

**Configuration Options:**

#### Split Mode
- **Single** (recommended): One HTTPRoute for all hostnames
  - ✅ Optimal for Gateway API
  - ✅ Efficient resource usage
  - ✅ Best when all hosts have same routing rules

- **Per-host**: Separate HTTPRoute per hostname
  - ✅ Maximum flexibility
  - ✅ Independent management per host
  - ✅ Best for different policies per host

- **Per-pattern**: Group by hostname patterns
  - ✅ Intelligent organization
  - ✅ Groups similar hosts (e.g., *.dev.example.com)
  - ✅ Best for large deployments

#### Gateway Name
- Name of the Gateway resource to reference
- Default derived from Ingress class name
- Press Enter to use default

#### Gateway Class
- Gateway class name (e.g., nginx, istio, contour)
- Default: nginx
- Press Enter to use default

**Tip**: Use defaults for most cases. They're derived intelligently from your Ingress configuration.

### Step 5: Preview Conversion

```
─────────────────────────────────────────────────────────────
Step 5: Preview Conversion
─────────────────────────────────────────────────────────────

✓ Generated 1 HTTPRoute(s)

Preview (first HTTPRoute):
───────────────────────────────────────────────────────────
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
    - name: app-service
      port: 80
      weight: 1
───────────────────────────────────────────────────────────

Press Enter to continue...
```

**What to review:**
- ✅ **Hostnames**: All expected hosts present?
- ✅ **Rules**: Correct paths and backends?
- ✅ **Timeouts**: Appropriate timeout values?
- ✅ **Backend refs**: Correct service names and ports?

**Tip**: This is your last chance to review before saving. Take your time!

### Step 6: Confirm and Save

```
─────────────────────────────────────────────────────────────
Step 6: Confirm and Save
─────────────────────────────────────────────────────────────

Validating HTTPRoute(s)...
✓ Validation passed

What would you like to do?
  1. Save to file
  2. Print to stdout
  3. Cancel

Select option [1-3]: 1

Output filename (default: my-app-ingress-httproute.yaml):
Filename (Enter for default): _

✓ Saved to: my-app-ingress-httproute.yaml
```

**Options:**

1. **Save to file**: Write HTTPRoute to a YAML file
   - Default filename: `<ingress-name>-httproute.yaml`
   - Customize filename if desired

2. **Print to stdout**: Display HTTPRoute in terminal
   - Good for piping to other commands
   - Copy/paste to apply directly

3. **Cancel**: Abort without saving
   - No files written
   - Can start over

**Tip**: Choose "Save to file" for most cases. It's easier to review and apply later.

## Example Session

### Complete Interactive Migration

```bash
$ ingress-to-gateway interactive

╔════════════════════════════════════════════════════════════╗
║   Welcome to ingress-to-gateway Interactive Migration!    ║
╚════════════════════════════════════════════════════════════╝

This wizard will guide you through migrating your Ingress
resources to Gateway API HTTPRoute step by step.

─────────────────────────────────────────────────────────────
Step 1: Select Namespace
─────────────────────────────────────────────────────────────

Current namespace: default

Options:
  1. Use current namespace
  2. List all namespaces
  3. Enter namespace manually

Select option [1-3]: 1

─────────────────────────────────────────────────────────────
Step 2: Select Ingress Resource
─────────────────────────────────────────────────────────────

Found 2 Ingress resource(s) in namespace 'default':

  1. webapp-ingress
     Class: nginx
     Hosts: 3 | Rules: 5 | TLS: true

  2. api-ingress
     Class: nginx
     Hosts: 1 | Rules: 2 | TLS: false

Select Ingress [1-2]: 1

─────────────────────────────────────────────────────────────
Step 3: Migration Analysis
─────────────────────────────────────────────────────────────

Ingress: default/webapp-ingress
Ingress Class: nginx
Hosts: 3 | Paths: 5 | TLS: true

✓ Detected Features:
  • URL_REWRITE
  • TLS_TERMINATION
  • PROXY_READ_TIMEOUT
  • CORS

✅ Migration Readiness: READY (Complexity: 12)

💡 Recommendations:
  • Use 'single' split mode for optimal Gateway API resource usage
  • Both timeouts.request and timeouts.backendRequest will be set
  • Ensure Gateway has matching HTTPS listeners configured

Press Enter to continue...

─────────────────────────────────────────────────────────────
Step 4: Configure Conversion Options
─────────────────────────────────────────────────────────────

Select split mode:
  1. Single   - One HTTPRoute for all hostnames (recommended)
  2. Per-host - Separate HTTPRoute per hostname
  3. Per-pattern - Group by hostname patterns

Split mode [1-3]: 1
✓ Split mode: single

Gateway name (default: gateway-nginx):
Gateway name (Enter for default):
✓ Gateway: gateway-nginx

Gateway class (default: nginx):
Gateway class (Enter for default):
✓ Gateway class: nginx

─────────────────────────────────────────────────────────────
Step 5: Preview Conversion
─────────────────────────────────────────────────────────────

✓ Generated 1 HTTPRoute(s)

Preview (first HTTPRoute):
───────────────────────────────────────────────────────────
[HTTPRoute YAML output...]
───────────────────────────────────────────────────────────

Press Enter to continue...

─────────────────────────────────────────────────────────────
Step 6: Confirm and Save
─────────────────────────────────────────────────────────────

Validating HTTPRoute(s)...
✓ Validation passed

What would you like to do?
  1. Save to file
  2. Print to stdout
  3. Cancel

Select option [1-3]: 1

Output filename (default: webapp-ingress-httproute.yaml):
Filename (Enter for default):

✓ Saved to: webapp-ingress-httproute.yaml

╔════════════════════════════════════════════════════════════╗
║              Migration Completed Successfully!            ║
╚════════════════════════════════════════════════════════════╝

Next steps:
  1. Review the generated HTTPRoute(s)
  2. Ensure Gateway resource exists in your cluster
  3. Apply: kubectl apply -f webapp-ingress-httproute.yaml
  4. Test traffic routing
  5. Monitor for any issues

Thank you for using ingress-to-gateway! 🚀
```

## Tips and Best Practices

### Before Starting

1. **Know your cluster**: Ensure you have the correct kubeconfig active
2. **List resources first**: Run `kubectl get ingress -A` to see what you have
3. **Check Gateway exists**: Verify Gateway resource before converting

### During the Wizard

1. **Read recommendations**: They're based on your specific Ingress
2. **Use defaults**: They're intelligently derived from your configuration
3. **Review preview carefully**: Last chance to catch issues
4. **Save to file**: Easier to review and apply later

### After Completion

1. **Review generated file**: Open in editor and verify
2. **Validate separately**: Run `ingress-to-gateway validate <file>`
3. **Test in staging first**: Don't apply directly to production
4. **Keep backups**: Save original Ingress YAML before deleting

### Common Workflows

#### Learning the Tool
```bash
# Run interactive mode multiple times
# Try different split modes
# Compare outputs
ingress-to-gateway interactive
```

#### Careful Migration
```bash
# Use interactive for analysis
ingress-to-gateway interactive

# Review generated file
cat webapp-ingress-httproute.yaml

# Validate
ingress-to-gateway validate webapp-ingress-httproute.yaml

# Apply
kubectl apply -f webapp-ingress-httproute.yaml
```

#### Exploratory Analysis
```bash
# Run once with single mode
ingress-to-gateway interactive
# Choose split mode: 1

# Run again with per-host mode
ingress-to-gateway interactive
# Choose split mode: 2

# Compare outputs
diff webapp-ingress-httproute.yaml webapp-ingress-httproute-per-host.yaml
```

## Troubleshooting

### "No Ingress resources found"

**Problem**: Wizard can't find Ingress in selected namespace

**Solutions**:
1. Check namespace: `kubectl get ns`
2. List Ingress: `kubectl get ingress -A`
3. Verify permissions: `kubectl auth can-i list ingress`

### "Failed to list namespaces"

**Problem**: Insufficient permissions

**Solutions**:
1. Check kubeconfig: `kubectl config view`
2. Verify context: `kubectl config current-context`
3. Check permissions: `kubectl auth can-i list namespaces`

### "Validation found errors"

**Problem**: Generated HTTPRoute has validation errors

**Solutions**:
1. Review error messages carefully
2. Check [Troubleshooting Guide](TROUBLESHOOTING.md)
3. Fix issues and run wizard again

### Can't input text

**Problem**: Terminal not accepting input

**Solutions**:
1. Check terminal type: `echo $TERM`
2. Use different terminal emulator
3. Ensure stdin is not redirected

## Keyboard Shortcuts

- **Enter**: Accept default / Continue
- **Ctrl+C**: Cancel wizard at any time
- **Ctrl+D**: EOF (same as cancel)

## Accessibility

The interactive wizard:
- ✅ Uses clear, numbered menus
- ✅ Provides default options
- ✅ Shows current selections
- ✅ Gives descriptive prompts
- ✅ Validates input

## Next Steps

After using interactive mode:

1. Try command-line mode: [Convert Command](API-REFERENCE.md#convert)
2. Batch convert multiple: [Batch Command](API-REFERENCE.md#batch)
3. Learn migration strategies: [Migration Strategies](MIGRATION-STRATEGIES.md)

## Feedback

Interactive mode not working as expected?
- Open issue: https://github.com/mayens/ingress-to-gateway/issues
- Include: Terminal type, OS, error messages

We're constantly improving the interactive experience based on user feedback!
