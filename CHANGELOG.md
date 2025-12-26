# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2025-12-26

### Major Release - Helm v4 Migration

This release migrates from Helm v3 to Helm v4, bringing compatibility with the latest Helm stable release. The migration includes updates to all dependencies and modernizes the build infrastructure.

### Changed

#### Major Updates

- **helm.sh/helm/v4**: Migrated from v3.19.4 to v4.0.0
  - Updated imports to use `helm.sh/helm/v4/pkg/repo/v1` (versioned repo package)
  - Updated chart handling to use `helm.sh/helm/v4/pkg/chart/v2`
  - Added type assertions for chart.Charter interface
  - All existing charts remain fully compatible

#### Dependencies

- **Go**: Updated from 1.24.0 to 1.25.0 with toolchain 1.25.5
- **cloud.google.com/go/storage**: v1.39.1 → v1.58.0
- **golang.org/x/oauth2**: v0.28.0 → v0.34.0
- **google.golang.org/api**: v0.227.0 → v0.258.0
- **Kubernetes libraries**: v0.33.3 → v0.35.0 (all k8s.io packages)
- **google.golang.org/grpc**: v1.71.0 → v1.78.0
- **google.golang.org/protobuf**: v1.36.5 → v1.36.11
- **OpenTelemetry**: Updated all go.opentelemetry.io packages to latest versions
- **golang.org/x packages**: Updated all to latest versions (crypto, net, sys, term, text, time, sync)
- Many other indirect dependencies updated for security and compatibility

#### CI/CD

- **GitHub Actions**: Updated all actions to latest versions
  - actions/checkout: v3 → v4
  - actions/setup-go: v3 → v5 (with Go 1.25)
  - github/codeql-action: v2 → v3
  - golangci/golangci-lint-action: v3 → v7
- Improved CI pipeline reliability and build performance

### Added

- Helm v4 support with backward compatibility for v1 and v2 charts
- Enhanced compatibility with latest Helm and Kubernetes versions
- Better support for modern Go toolchain features
- Comprehensive dependency updates ensuring latest security patches

### Security

- All dependencies updated to latest versions with security patches
- Updated to Go 1.25 with latest security improvements
- Continued Trivy vulnerability scanning in CI pipeline

### Compatibility

- **Helm v4 Compatible**: Fully compatible with Helm 4.0+
- **Chart Compatibility**: Supports apiVersion v1, v2, and v3 charts
- Tested with Go 1.25+
- Compatible with Kubernetes 0.35.0+
- Works with latest Google Cloud Storage API

### Migration Notes

- **No breaking changes for end users**: All existing `gs://` repositories and charts work without modification
- **Plugin interface unchanged**: All helm-gcs commands work exactly as before
- **Chart format unchanged**: v1 and v2 charts continue to work seamlessly
- Recommended for all users to upgrade for Helm v4 compatibility and improved security

## [0.4.2] - Previous Release

For changes in versions prior to 1.0.0, please see the [GitHub Releases](https://github.com/hayorov/helm-gcs/releases) page.
