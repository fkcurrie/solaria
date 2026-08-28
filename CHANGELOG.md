# Changelog

All notable changes to the Solaria project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

---

## [v0.9.0-alpha.1] - 2026-08-28

### Added
- Multi-tiered site location resolver in pkg/location/resolver.go supporting hardware GPS (gpsd), IP geolocation fallback, environment overrides, and safe defaults.
- Automated multi-linter suite (.golangci.yml) and GitHub Actions workflow (.github/workflows/golangci-lint.yml).
- Automated PR Code Reviewer and Security Bot (.github/workflows/pr-bot-reviewer.yml) with support for on-demand /review slash command triggers (.github/workflows/slash-command-reviewer.yml).
- Dependabot automated dependency scanning configuration (.github/dependabot.yml).
- Standardized pull request quality checklist template (.github/pull_request_template.md).
- Systemd service unit files for bridge, edge, and cloud daemons (systemd/solaria-bridge.service, systemd/solaria-edge.service, systemd/solaria-cloud.service).
- Formalized Enterprise Release Protocol specification (docs/developer/release-protocol.md).

### Changed
- Removed legacy hardcoded site location coordinates and array peak wattage across setup.sh, install.sh, cmd/bridge/main.go, cmd/cloud-server/main.go, and HTML templates.
- Enhanced .github/workflows/bootstrap-ci.yml to run 31-probe E2E system audit automatically on every commit.
- Upgraded release workflow (.github/workflows/release.yml) to enforce 6 release quality gates before generating multi-architecture tarballs, zips, checksums, and draft releases.

### Fixed
- Fixed static location coordinate coupling in cloud server astronomical sun calculations and diagnostic bundle responses.
