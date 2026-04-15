<p align="center">
	<img src="https://raw.githubusercontent.com/hayorov/helm-gcs/master/assets/helm-gcs-logo.png" alt="helm-gcs logo" width="400"/>
</p>

<h1 align="center">helm-gcs</h1>

<p align="center">
  <strong>Helm plugin for managing chart repositories on Google Cloud Storage</strong>
</p>

<p align="center">
  <a href="https://github.com/hayorov/helm-gcs/releases/latest">
    <img src="https://img.shields.io/github/v/release/hayorov/helm-gcs?style=flat-square" alt="Latest Release"/>
  </a>
  <a href="https://github.com/hayorov/helm-gcs/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/hayorov/helm-gcs/test.yml?style=flat-square" alt="Build Status"/>
  </a>
  <a href="https://github.com/hayorov/helm-gcs/blob/master/LICENSE">
    <img src="https://img.shields.io/github/license/hayorov/helm-gcs?style=flat-square" alt="License"/>
  </a>
  <a href="https://goreportcard.com/report/github.com/hayorov/helm-gcs">
    <img src="https://goreportcard.com/badge/github.com/hayorov/helm-gcs?style=flat-square" alt="Go Report Card"/>
  </a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Helm%204-supported-success?style=flat-square&logo=helm" alt="Helm 4"/>
  <img src="https://img.shields.io/badge/Helm%203-supported-success?style=flat-square&logo=helm" alt="Helm 3"/>
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go Version"/>
  <img src="https://img.shields.io/badge/GCP-Cloud%20Storage-4285F4?style=flat-square&logo=googlecloud" alt="Google Cloud Storage"/>
</p>

---

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Authentication](#-authentication)
- [Usage](#-usage)
  - [Initialize Repository](#initialize-repository)
  - [Push Charts](#push-charts)
  - [Remove Charts](#remove-charts)
- [Advanced Features](#-advanced-features)
- [Troubleshooting](#-troubleshooting)
- [Version Compatibility](#-version-compatibility)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🎯 Overview

**helm-gcs** is a [Helm](https://helm.sh/) plugin that enables you to manage private Helm chart repositories using [Google Cloud Storage](https://cloud.google.com/storage/) (GCS) buckets as the backend storage.

Store, version, and distribute your Helm charts on GCS with the same ease and security you expect from Google Cloud Platform.

### Why helm-gcs?

- **🔐 Secure**: Leverage GCP IAM for fine-grained access control
- **💰 Cost-effective**: Pay only for storage used, no infrastructure to maintain
- **🚀 Fast**: Benefit from Google's global CDN and low-latency storage
- **🔄 Concurrent-safe**: Built-in optimistic locking prevents race conditions
- **📦 Simple**: Works seamlessly with existing Helm workflows
- **☁️ Cloud-native**: Native integration with Google Cloud Platform

---

## ✨ Features

- 📥 **Push/Pull charts** to/from GCS buckets
- 🔧 **Initialize repositories** anywhere in your GCS bucket
- 🗑️ **Remove charts** by version or entirely
- 🔐 **Multiple authentication methods** (ADC, Service Account, OAuth)
- 🔄 **Concurrent update handling** with automatic retry
- 🏷️ **Custom metadata** support for chart objects
- 📁 **Bucket path organization** for structured chart storage
- 🌍 **Multi-platform support** (Linux, macOS, Windows on amd64/arm64)
- ✅ **Helm 4 compatible** (also supports Helm 3)

---

## 📦 Installation

The install source is the same for Helm 3 and Helm 4:

- Helm 4 installs the native `gcs` + `gcs-getter` plugin pair automatically
- Helm 3 installs the legacy single-plugin layout automatically

### Install Latest

```bash
# Helm 4
helm plugin install https://github.com/hayorov/helm-gcs.git --verify=false

# Helm 3
helm plugin install https://github.com/hayorov/helm-gcs.git
```

This installs:
- Helm 4: `gcs` (CLI) and `gcs-getter` (Getter)
- Helm 3: `gcs` (single combined plugin)

Verify installation:

```bash
helm plugin list
helm gcs version
```

### Install Specific Version

```bash
# Helm 4
helm plugin install https://github.com/hayorov/helm-gcs.git --version 0.7.0 --verify=false

# Helm 3
helm plugin install https://github.com/hayorov/helm-gcs.git --version 0.7.0
```

### Update to Latest

```bash
# Helm 4
helm plugin update gcs
helm plugin update gcs-getter

# Helm 3
helm plugin update gcs
```

### Verify Installation

```bash
helm gcs version
```

---

## 🚀 Quick Start

Get started in under 2 minutes:

```bash
# 1. Initialize a new repository in your GCS bucket
helm gcs init gs://my-bucket/helm-charts

# 2. Add your repository to Helm
helm repo add my-repo gs://my-bucket/helm-charts

# 3. Package your chart
helm package ./my-chart

# 4. Push chart to your repository
helm gcs push my-chart-1.0.0.tgz my-repo

# 5. Update Helm cache
helm repo update

# 6. Search for your chart
helm search repo my-repo

# 7. Install your chart
helm install my-release my-repo/my-chart
```

---

## 🔐 Authentication

helm-gcs supports multiple authentication methods (in priority order):

### 1. OAuth Access Token (Temporary)

```bash
export GOOGLE_OAUTH_ACCESS_TOKEN=$(gcloud auth print-access-token)
helm gcs push chart.tgz my-repo
```

> ⏱️ Token expires in 1 hour. Best for temporary operations.

### 2. Service Account Key File

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
helm gcs push chart.tgz my-repo
```

> 🔑 Recommended for CI/CD environments.

### 3. Application Default Credentials (ADC)

```bash
gcloud auth application-default login
helm gcs push chart.tgz my-repo
```

> 👤 Best for local development.

### Required IAM Permissions

Your service account or user needs these permissions:
- `storage.objects.get`
- `storage.objects.create`
- `storage.objects.delete`
- `storage.objects.list`

**Recommended IAM Role**: `Storage Object Admin` or `Storage Admin`

---

## 📖 Usage

### Initialize Repository

Create a new Helm repository in your GCS bucket:

```bash
helm gcs init gs://your-bucket/path/to/charts
```

**Options:**
- Repository can be created anywhere in your bucket
- Creates an empty `index.yaml` if it doesn't exist
- Safe to run multiple times (idempotent)

**Example with nested path:**

```bash
helm gcs init gs://company-charts/production/stable
```

### Add Repository to Helm

```bash
helm repo add stable-charts gs://company-charts/production/stable
helm repo add dev-charts gs://company-charts/development
```

Verify repositories:

```bash
helm repo list
```

### Push Charts

#### Basic Push

```bash
# Package your chart
helm package ./my-application

# Push to repository
helm gcs push my-application-1.0.0.tgz stable-charts
```

#### Push with Retry (Recommended for CI/CD)

```bash
helm gcs push my-application-1.0.0.tgz stable-charts --retry
```

> 🔄 Automatically retries if concurrent updates detected

#### Push with Custom Metadata

Add custom metadata to your chart object:

```bash
helm gcs push my-app-1.0.0.tgz stable-charts \
  --metadata env=production,team=platform,region=us-central1
```

#### Push to Bucket Path

Organize charts within your bucket:

```bash
helm gcs push my-app-1.0.0.tgz stable-charts --bucketPath=applications/backend
```

This stores the chart at: `gs://your-bucket/charts/applications/backend/my-app-1.0.0.tgz`

#### Force Push

Overwrite existing chart:

```bash
helm gcs push my-app-1.0.0.tgz stable-charts --force
```

> ⚠️ Use with caution - overwrites existing chart with same version

#### Push with Public Access

Make chart publicly accessible:

```bash
helm gcs push my-app-1.0.0.tgz stable-charts --public
```

### Remove Charts

#### Remove Specific Version

```bash
helm gcs remove my-application stable-charts --version 1.0.0
```

#### Remove All Versions

```bash
helm gcs remove my-application stable-charts
```

> 💡 Don't forget to update your local cache: `helm repo update`

---

## 🔧 Advanced Features

### Concurrent Updates

helm-gcs uses optimistic locking to prevent index corruption during concurrent updates:

```bash
# If you see: "Error: index is out-of-date"
# Simply retry the command or use --retry flag

helm gcs push chart.tgz my-repo --retry
```

The plugin will automatically:
1. Detect concurrent modification
2. Fetch latest index
3. Retry the operation
4. Use exponential backoff

### Debug Mode

Enable detailed logging:

```bash
# Using environment variable
export HELM_GCS_DEBUG=true
helm gcs push chart.tgz my-repo

# Or use global flag
helm gcs push chart.tgz my-repo --debug
```

### Custom Repository URL

Use custom domain or CDN:

```bash
helm gcs push chart.tgz my-repo \
  --public \
  --publicURL=https://charts.example.com
```

---

## 🔍 Troubleshooting

### Common Issues

#### Authentication Errors

```
Error: failed to authenticate to GCS
```

**Solution:**
1. Verify credentials: `gcloud auth list`
2. Check `GOOGLE_APPLICATION_CREDENTIALS` path
3. Ensure service account has required permissions
4. Try: `gcloud auth application-default login`

#### Index Out of Date

```
Error: update index file: index is out-of-date
```

**Solution:** Use `--retry` flag for automatic retry:
```bash
helm gcs push chart.tgz my-repo --retry
```

#### Permission Denied

```
Error: googleapi: Error 403: Forbidden
```

**Solution:**
1. Verify IAM permissions (need `Storage Object Admin`)
2. Check bucket name is correct
3. Ensure bucket exists: `gsutil ls gs://your-bucket`

#### Chart Already Exists

```
Error: chart already indexed
```

**Solution:** Use `--force` to overwrite:
```bash
helm gcs push chart.tgz my-repo --force
```

### Enable Debug Logging

```bash
export HELM_GCS_DEBUG=true
helm gcs push chart.tgz my-repo --debug
```

### Get Help

```bash
helm gcs --help
helm gcs push --help
```

---

## 📊 Version Compatibility

| helm-gcs Version | Helm Version | Go Version | Notes | Status |
|------------------|--------------|------------|-------|--------|
| 0.7.x | Helm 4.x (native) | 1.25+ | Two separate plugins (CLI + Getter) | ✅ Active |
| 0.7.x | Helm 3.x (legacy) | 1.25+ | Single combined plugin | ✅ Active |
| 0.6.x | Helm 4.x, 3.x | 1.25+ | Legacy mode on Helm 4 | ✅ Supported |
| 0.5.x | Helm 3.x | 1.24+ | | ✅ Supported |
| 0.4.x | Helm 3.x | 1.20+ | | ⚠️ Deprecated |
| 0.3.x | Helm 3.x | 1.16+ | | ⚠️ Deprecated |
| 0.2.x | Helm 2.x | 1.13+ | | ❌ Unsupported |

### Helm 4 Architecture

Helm 4 enforces a strict "one plugin = one type" model. A single plugin cannot be both a CLI plugin and a Getter plugin. helm-gcs 0.7.0+ keeps the install command the same as Helm 3, but installs two separate plugins automatically on Helm 4:

| Plugin | Type | Purpose |
|--------|------|---------|
| `gcs` | `cli/v1` | `helm gcs init/push/rm` commands |
| `gcs-getter` | `getter/v1` | `gs://` protocol for `helm repo add`, `helm pull` |

### Helm 3 Compatibility

Helm 3 uses a combined plugin model where one plugin handles both CLI commands and protocol downloading. helm-gcs 0.7.x keeps full backward compatibility while sharing the same repository install URL as Helm 4:

```bash
# Install for Helm 3
helm plugin install https://github.com/hayorov/helm-gcs.git

# All features work with single plugin
helm gcs init gs://bucket/charts      # CLI commands
helm repo add myrepo gs://bucket/charts  # gs:// protocol
```

The repository root `plugin.yaml` is used as a bootstrap entrypoint for both Helm 3 and Helm 4:
- Helm 3 keeps the legacy combined plugin layout
- Helm 4 installs the native `gcs` and `gcs-getter` plugin pair

### Helm 2 Users

For Helm 2 support, use version 0.2.2:

```bash
helm plugin install https://github.com/hayorov/helm-gcs.git --version 0.2.2
```

> ⚠️ Helm 2 reached end-of-life. Please upgrade to Helm 3 or 4.

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/hayorov/helm-gcs.git
cd helm-gcs

# Copy environment template
cp .env.example .env
# Edit .env with your GCS test bucket and credentials

# Run tests
go test -v ./...

# Run integration tests (requires GCS credentials)
go test -v -tags=integration ./pkg/repo

# Build
go build -o bin/helm-gcs ./cmd/helm-gcs
```

### Running Tests

```bash
# Unit tests
go test -v -race ./...

# Integration tests (requires GCS bucket)
export GCS_TEST_BUCKET=gs://your-test-bucket/helm-gcs-tests
go test -v -tags=integration ./pkg/repo

# With debug logging
export HELM_GCS_DEBUG=true
go test -v -tags=integration ./pkg/repo
```

### Code Quality

```bash
# Format code
gofmt -s -w .

# Run linter
golangci-lint run

# Check code complexity
gocyclo -over 19 cmd pkg

# Vet code
go vet ./...
```

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Helm](https://helm.sh/) - The package manager for Kubernetes
- [Google Cloud Storage](https://cloud.google.com/storage/) - Object storage service
- All our [contributors](https://github.com/hayorov/helm-gcs/graphs/contributors)

---

## 📞 Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/hayorov/helm-gcs/issues)
- 💬 **Questions**: [GitHub Discussions](https://github.com/hayorov/helm-gcs/discussions)
- 📖 **Documentation**: [Project Wiki](https://github.com/hayorov/helm-gcs/wiki)

---

<p align="center">
  Made with ❤️ by the helm-gcs community
</p>

<p align="center">
  <a href="https://github.com/hayorov/helm-gcs/stargazers">⭐ Star us on GitHub</a> •
  <a href="https://github.com/hayorov/helm-gcs/issues">🐛 Report Bug</a> •
  <a href="https://github.com/hayorov/helm-gcs/issues">✨ Request Feature</a>
</p>
