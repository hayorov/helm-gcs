# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2025-12-25

### Minor Release

This release updates all dependencies to their latest versions and modernizes the build infrastructure.

### Changed

#### Dependencies
- **Go**: Updated from 1.24.0 to 1.25.0 with toolchain 1.25.5
- **cloud.google.com/go/storage**: v1.39.1 → v1.58.0
- **golang.org/x/oauth2**: v0.28.0 → v0.34.0
- **google.golang.org/api**: v0.227.0 → v0.258.0
- **helm.sh/helm/v3**: v3.18.5 → v3.19.4
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
- Comprehensive dependency updates ensuring latest security patches
- Enhanced compatibility with latest Helm and Kubernetes versions
- Better support for modern Go toolchain features

### Security
- All dependencies updated to latest versions with security patches
- Updated to Go 1.25 with latest security improvements
- Continued Trivy vulnerability scanning in CI pipeline

### Compatibility
- Fully compatible with Helm 3.19+
- Tested with Go 1.25+
- Compatible with Kubernetes 0.35.0+
- Works with latest Google Cloud Storage API

### Notes
- No breaking changes from 0.4.x series
- All existing repositories and charts remain fully compatible
- Recommended for all users to upgrade for improved security and performance

## [0.4.2] - Previous Release

For changes in versions prior to 1.0.0, please see the [GitHub Releases](https://github.com/hayorov/helm-gcs/releases) page.
