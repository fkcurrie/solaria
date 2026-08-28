# Solaria Enterprise Release Protocol Specification

This document defines the official, gate-driven release protocol for Solaria. Every production and pre-release tag MUST pass all six release quality gates before distribution artifacts are published.

---

## 1. Versioning Scheme

Solaria follows Semantic Versioning (SemVer 2.0.0):

- **Major Versions (`vX.0.0`):** Breaking API contract changes or fundamental architectural shifts.
- **Minor Versions (`v0.Y.0` / `v1.Y.0`):** New backwards-compatible features, protocol engines, or major subsystem additions.
- **Patch Versions (`v0.Y.Z`):** Backwards-compatible bug fixes, security patches, or documentation updates.
- **Pre-Releases (`vX.Y.Z-alpha.N` / `vX.Y.Z-beta.N`):** Feature-complete release candidates for staging and testing.

---

## 2. The Six Release Quality Gates

Before any release tag (`v*`) is created, the code MUST pass the following six automated release quality gates:

```text
[Tag Pushed] --> [Gate 1: Code Linting]
             --> [Gate 2: Unit Test Suite]
             --> [Gate 3: 31-Probe E2E System Audit]
             --> [Gate 4: Vulnerability Database Scan]
             --> [Gate 5: Documentation Parity]
             --> [Gate 6: Multi-Arch Packaging] --> [GitHub Draft Release]
```

### Gate 1: Code Linting & Static Security Analysis
- Tool: `golangci-lint run ./...`
- Criteria: Zero unhandled errors, zero formatting violations, and zero security warnings.

### Gate 2: Package Unit Test Suite
- Tool: `go test -v ./...`
- Criteria: 100% test pass rate across all packages (`cmd/bridge`, `cmd/edge-agent`, `cmd/cloud-server`, `cmd/sre-agent`, `pkg/location`).

### Gate 3: 31-Probe E2E System Audit
- Tool: `bin/solaria-e2e-audit`
- Criteria: 31 out of 31 E2E probes pass against live background test daemons.

### Gate 4: Security Vulnerability Scan
- Tool: `govulncheck ./...`
- Criteria: Zero known CVE vulnerabilities in Go standard library or imported dependencies.

### Gate 5: Documentation Parity & Alignment
- Tool: Internal docs audit and markdown verification.
- Criteria: All installation flags, environment variables, and architecture diagrams match current codebase logic.

### Gate 6: Multi-Architecture Packaging & Checksum Verification
- Targets: `linux/amd64`, `linux/arm64`, `linux/arm` (v7).
- Deliverables: `.tar.gz` and `.zip` archives per architecture containing binaries, systemd service units, setup scripts, and `.env.example`.
- Manifest: Master `SHA256SUMS` file generated and verified.

---

## 3. Release Asset Structure

Standard directory layout inside every release archive (`solaria_vX.Y.Z_linux_ARCH.tar.gz` and `.zip`):

```text
solaria_vX.Y.Z_linux_ARCH/
├── bin/
│   ├── solaria-bridge
│   ├── solaria-edge
│   ├── solaria-cloud
│   └── solaria-e2e-audit
├── systemd/
│   ├── solaria-bridge.service
│   ├── solaria-edge.service
│   └── solaria-cloud.service
├── .env.example
├── install.sh
└── setup.sh
```

---

## 4. GitHub Release Workflow & Human Sign-Off

1. The automated 6-Gate Release Gatekeeper workflow (`.github/workflows/release.yml`) executes when a version tag (`v*`) is pushed.
2. Upon passing all six gates, the workflow attaches all compiled `.tar.gz`, `.zip`, and `SHA256SUMS` files to a **GitHub Draft Release**.
3. The release notes are populated automatically from `RELEASE_NOTES/vX.Y.Z.md`.
4. Release manager inspects the Draft Release and clicks **Publish Release**.
