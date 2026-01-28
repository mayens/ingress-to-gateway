# Documentation

Complete documentation for ingress-to-gateway - The comprehensive Ingress-NGINX to Gateway API migration tool.

## Quick Links

- 🚀 [Getting Started Guide](GETTING-STARTED.md) - Step-by-step guide for your first migration
- 📖 [API Reference](API-REFERENCE.md) - Complete CLI reference
- 🔄 [Annotation Mapping](ANNOTATION-MAPPING.md) - How NGINX annotations convert to Gateway API
- 📋 [Migration Strategies](MIGRATION-STRATEGIES.md) - Proven approaches for different scenarios
- ⏱️ [Timeout Configuration](TIMEOUT-CONFIGURATION.md) - Understanding timeout mappings
- 🔧 [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions

## Documentation Overview

### For First-Time Users

Start here if you're new to ingress-to-gateway or Gateway API migration:

1. **[Getting Started Guide](GETTING-STARTED.md)**
   - Prerequisites and installation
   - Your first migration walkthrough
   - Understanding the workflow
   - Common scenarios
   - Best practices

### For Reference

Use these when you need specific information:

2. **[API Reference](API-REFERENCE.md)**
   - Complete command-line interface documentation
   - All commands, flags, and options
   - Configuration file format
   - Environment variables
   - Exit codes

3. **[Annotation Mapping](ANNOTATION-MAPPING.md)**
   - Complete mapping of 17+ NGINX Ingress annotations
   - Gateway API equivalents
   - Conversion status for each annotation
   - Examples for each mapping
   - Unsupported features and workarounds

### For Planning

Consult these before starting your migration:

4. **[Migration Strategies](MIGRATION-STRATEGIES.md)**
   - 5 proven migration strategies
   - Strategy comparison table
   - When to use each approach
   - Step-by-step implementation guides
   - Risk mitigation techniques
   - Rollback plans

5. **[Timeout Configuration](TIMEOUT-CONFIGURATION.md)**
   - Understanding NGINX vs Gateway API timeouts
   - Why both `request` and `backendRequest` fields
   - Timeout constraint rules
   - Mapping guide with examples
   - Best practices
   - Common scenarios

### For Problem Solving

Use when you encounter issues:

6. **[Troubleshooting](TROUBLESHOOTING.md)**
   - Installation issues
   - Conversion issues
   - Validation errors
   - Runtime issues
   - Performance issues
   - Where to get help

## Documentation by Use Case

### "I want to migrate my first Ingress"

1. Read: [Getting Started Guide](GETTING-STARTED.md) - "Your First Migration" section
2. Run: `ingress-to-gateway audit`
3. Run: `ingress-to-gateway convert my-ingress -n default`
4. Read: [Troubleshooting](TROUBLESHOOTING.md) if issues occur

### "I need to understand timeout configuration"

1. Read: [Timeout Configuration](TIMEOUT-CONFIGURATION.md)
2. Check: [Annotation Mapping](ANNOTATION-MAPPING.md) - "Timeouts" section
3. Reference: [API Reference](API-REFERENCE.md) - `convert` command

### "I want to migrate a production system"

1. Read: [Migration Strategies](MIGRATION-STRATEGIES.md) - Choose appropriate strategy
2. Read: [Getting Started Guide](GETTING-STARTED.md) - Best practices
3. Use: [Troubleshooting](TROUBLESHOOTING.md) - Risk mitigation checklist
4. Reference: [API Reference](API-REFERENCE.md) - All commands

### "An annotation isn't converting"

1. Check: [Annotation Mapping](ANNOTATION-MAPPING.md) - Find your annotation
2. Check: Status (✅ Fully Supported, ⚠️ Partially Supported, ❌ Not Supported)
3. If unsupported: See workarounds in the same document
4. If issues: [Troubleshooting](TROUBLESHOOTING.md) - "Conversion Issues"

### "The tool isn't working"

1. Start: [Troubleshooting](TROUBLESHOOTING.md) - "Installation Issues"
2. Reference: [API Reference](API-REFERENCE.md) - Verify command syntax
3. If stuck: "Getting Help" section in Troubleshooting

### "I need command reference"

Go to: [API Reference](API-REFERENCE.md) - Find your command

## Quick Reference

### Most Common Commands

```bash
# Audit Ingress resources
ingress-to-gateway audit --all-namespaces --detailed

# Convert single Ingress
ingress-to-gateway convert my-ingress -n default -o httproute.yaml

# Batch convert all
ingress-to-gateway batch --all-namespaces -o httproutes/

# Validate HTTPRoute
ingress-to-gateway validate httproute.yaml
```

See [API Reference](API-REFERENCE.md) for complete command documentation.

### Most Important Concepts

1. **Rule Deduplication**: HTTPRoute rules apply to ALL hostnames in the `hostnames` list
   - See: [Getting Started Guide](GETTING-STARTED.md)

2. **Both Timeout Fields**: Always set both `request` and `backendRequest`
   - See: [Timeout Configuration](TIMEOUT-CONFIGURATION.md)

3. **Split Modes**: Three strategies for organizing HTTPRoutes
   - See: [API Reference](API-REFERENCE.md) - `convert` command

4. **Validation is Critical**: Always validate before applying
   - See: [Getting Started Guide](GETTING-STARTED.md) - Best practices

## Document Summaries

### Getting Started Guide (Complete Walkthrough)

**Length**: ~400 lines

**Contents**:
- Prerequisites checklist
- Installation instructions (3 methods)
- 10-step migration walkthrough
- 5 common scenarios with examples
- 7 best practices
- Troubleshooting quick tips

**Best for**: First-time users, complete migrations

---

### API Reference (Command Documentation)

**Length**: ~700 lines

**Contents**:
- Global flags
- 5 commands with all flags and options
- Exit codes
- Configuration file format
- Environment variables
- 10+ examples

**Best for**: Command syntax lookup, automation scripts

---

### Annotation Mapping (Feature Conversions)

**Length**: ~800 lines

**Contents**:
- 20+ annotation mappings
- Status indicators (✅ ⚠️ ❌ 🔍)
- Before/after examples for each
- Gateway API equivalents
- Workarounds for unsupported features
- Summary comparison table

**Best for**: Understanding what converts and how

---

### Migration Strategies (Proven Approaches)

**Length**: ~800 lines

**Contents**:
- 5 migration strategies detailed
- Strategy comparison table
- Decision tree for choosing strategy
- Implementation steps for each
- Risk mitigation techniques
- Monitoring and rollback plans

**Best for**: Planning production migrations

---

### Timeout Configuration (Deep Dive)

**Length**: ~600 lines

**Contents**:
- NGINX timeout annotations explained
- Gateway API timeout fields explained
- Mapping guide with 5 scenarios
- Why both fields are set
- Constraint rules
- 5 common scenarios
- Troubleshooting timeout issues

**Best for**: Understanding timeout behavior

---

### Troubleshooting (Problem Solving)

**Length**: ~600 lines

**Contents**:
- Installation issues (3 problems)
- Conversion issues (4 problems)
- Validation errors (6 problems)
- Runtime issues (6 problems)
- Performance issues (2 problems)
- How to get help
- Quick reference commands

**Best for**: Solving specific problems

---

## Additional Resources

### In Repository

- [README.md](../README.md) - Project overview and quick start
- [QUICKSTART.md](../QUICKSTART.md) - Quick start guide
- [CONTRIBUTING.md](../CONTRIBUTING.md) - How to contribute
- [PROJECT-SUMMARY.md](../PROJECT-SUMMARY.md) - Complete project summary
- [LICENSE](../LICENSE) - Apache 2.0 license

### Related Documentation (from bash project)

Located in `/home/mayens/test/audit-ingress/`:

- `MIGRATION-TABLE.md` - Official Gateway API mapping table
- `SPLITTING-GUIDE.md` - HTTPRoute splitting strategies explained
- `DEDUPLICATION-FIX.md` - Rule deduplication deep dive
- `TIMEOUT-MAPPING.md` - Timeout field mapping details
- `TIMEOUT-BEST-PRACTICES.md` - Why both timeout fields
- `COMPETITIVE-ANALYSIS.md` - Comparison with ingress2gateway
- `IMPROVEMENTS-SUMMARY.md` - All fixes applied based on user feedback

### External Resources

- [Gateway API Official Documentation](https://gateway-api.sigs.k8s.io/)
- [NGINX Ingress Controller Documentation](https://kubernetes.github.io/ingress-nginx/)
- [Gateway API GitHub](https://github.com/kubernetes-sigs/gateway-api)
- [ingress2gateway GitHub](https://github.com/kubernetes-sigs/ingress2gateway)

## Documentation Metrics

- **Total documents**: 6 comprehensive guides
- **Total lines**: ~4,000 lines of documentation
- **Total examples**: 100+ code examples
- **Total scenarios**: 30+ real-world scenarios covered

## Contributing to Documentation

Found an issue or want to improve the documentation?

1. Open an issue: https://github.com/mayens/ingress-to-gateway/issues
2. Submit a pull request
3. See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines

## Documentation License

All documentation is licensed under Apache License 2.0, same as the code.

---

**Happy Migrating!** 🚀

Start with the [Getting Started Guide](GETTING-STARTED.md) and let us know how it goes!
