# Solaria Automated CI/CD Pipelines & Release Workflows

Solaria uses GitHub Actions to enforce rigorous multi-architecture compilation, static analysis linting, security scanning, 31-probe E2E auditing, and automated release packaging.

---

## 🛠️ GitHub Actions Workflows Summary

### 1. Multi-Arch Binary Compilation (`.github/workflows/ci.yml`)
Builds Go binaries across supported host architectures on every commit:
- `linux/amd64` (Standard x86_64 servers & Cloud Run)
- `linux/arm64` (Raspberry Pi 3/4/5, Apple Silicon)
- `linux/arm/v7` (Raspberry Pi 2, Zero 2W)
- `darwin/arm64` (macOS Apple Silicon)
- `darwin/amd64` (macOS Intel)

### 2. Multi-Distro QEMU Matrix & 31-Probe E2E Audit (`.github/workflows/bootstrap-ci.yml`)
Tests the `setup.sh` installer inside real Docker containers and executes `bin/solaria-e2e-audit` across 31 subsystem probes:
- `debian:bookworm-slim` (ARM64 via QEMU)
- `debian:bookworm-slim` (x86_64)
- `ubuntu:24.04` (x86_64)

### 3. GolangCI-Lint Quality & Standards Check (`.github/workflows/golangci-lint.yml`)
Executes 10 Go linters (`errcheck`, `gofmt`, `gosec`, `govet`, `ineffassign`, `staticcheck`, `unused`, `misspell`, `bodyclose`, `copyloopvar`) configured in `.golangci.yml` on every push and PR.

### 4. PR Bot Security Reviewer & Slash Command Trigger (`.github/workflows/pr-bot-reviewer.yml` & `.github/workflows/slash-command-reviewer.yml`)
- Runs `govulncheck`, `gosec`, `gofmt`, and `go test` on every Pull Request, posting interactive review scorecards and applying quality labels.
- Responds on-demand when contributors comment `/review` or `/audit` on a Pull Request.

### 5. Dependabot Automated Dependency Security (`.github/dependabot.yml`)
Performs weekly automated vulnerability scans and opens update PRs for Go modules (`go.mod`) and GitHub Actions workflows.

### 6. Production 6-Gate Release Engine (`.github/workflows/release.yml`)
Enforces the 6 release quality gates whenever a version tag (`v*`) is pushed, producing `.tar.gz` and `.zip` bundles (`amd64`, `arm64`, `armv7`), `SHA256SUMS` manifests, and generating a GitHub Draft Release.
