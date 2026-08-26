# Multi-Architecture CI Workflows & QEMU

Solaria uses GitHub Actions to enforce rigorous multi-architecture compilation and containerized OS compatibility testing.

---

## 🛠️ GitHub Actions Workflows

### 1. Multi-Arch Binary Compilation (`.github/workflows/ci.yml`)
Every commit builds Go binaries for:
- `linux/amd64` (Standard x86_64 servers & Cloud Run)
- `linux/arm64` (Raspberry Pi 3/4/5, Apple Silicon)
- `linux/arm/v7` (Raspberry Pi 2, Zero 2W)
- `darwin/arm64` (macOS Apple Silicon)
- `darwin/amd64` (macOS Intel)

### 2. Multi-Distro QEMU Bootstrap Matrix (`.github/workflows/bootstrap-ci.yml`)
Tests the `setup.sh` installer inside real Docker containers:
- `debian:bookworm-slim` (ARM64 via QEMU)
- `debian:bookworm-slim` (x86_64)
- `ubuntu:24.04` (x86_64)
