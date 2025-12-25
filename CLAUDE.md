# CLAUDE.md - Project Guide for helm-gcs

## Project Overview

**helm-gcs** is a Helm plugin that enables managing private Helm chart repositories using Google Cloud Storage (GCS) as the backend storage. It allows developers to store, push, pull, and manage Helm charts in GCS buckets instead of traditional HTTP/S registries.

**Repository**: https://github.com/hayorov/helm-gcs
**Current Version**: 0.4.2
**Language**: Go 1.24+ (toolchain 1.25.1)
**License**: MIT

### Key Features
- Initialize Helm repositories in GCS buckets
- Push/pull Helm charts to/from GCS
- Support for both private and public GCS buckets
- Multiple authentication methods (ADC, service account, OAuth token)
- Concurrent update handling with optimistic locking
- Cross-platform support (Linux, macOS, Windows on amd64/arm64)

---

## Architecture Overview

### Layered Architecture
```
┌─────────────────────────────────────┐
│   CLI Layer (Cobra Commands)       │  cmd/helm-gcs/cmd/*.go
├─────────────────────────────────────┤
│   Domain Layer (Repo Management)   │  pkg/repo/repo.go
├─────────────────────────────────────┤
│   Infrastructure (GCS Client)      │  pkg/gcs/gcs.go
├─────────────────────────────────────┤
│   External Services (GCS API)      │  cloud.google.com/go/storage
└─────────────────────────────────────┘
```

### Directory Structure
```
helm-gcs/
├── cmd/helm-gcs/              # CLI entry point and commands
│   ├── main.go               # Simple main() that calls cmd.Execute()
│   └── cmd/                  # Cobra command definitions
│       ├── root.go           # Root command with GCS client init
│       ├── init.go           # Initialize repo command
│       ├── push.go           # Push chart command
│       ├── pull.go           # Pull chart command (Helm integration)
│       ├── rm.go             # Remove chart command
│       └── version.go        # Version command
├── pkg/                      # Core business logic
│   ├── gcs/                  # GCS client wrapper (53 lines)
│   └── repo/                 # Repository operations (408 lines - main logic)
├── scripts/                  # Helper scripts
│   ├── install.sh           # Plugin installation
│   └── pull.sh              # Wrapper for pull command
├── plugin.yaml              # Helm plugin manifest
└── .github/workflows/       # CI/CD pipelines
```

---

## Key Files Reference

### Entry Points
- **cmd/helm-gcs/main.go** - Simple main() that delegates to Cobra
- **cmd/helm-gcs/cmd/root.go:15** - CLI initialization and GCS client setup

### Core Business Logic
- **pkg/repo/repo.go** (408 lines) - Main repository management logic
  - `Repo` struct - Manages repository operations
  - `Create()` - Initialize new repository
  - `PushChart()` - Upload chart with index update
  - `RemoveChart()` - Delete chart versions
  - `uploadIndexFile()` - Update Helm index with optimistic locking
  - `indexFile()` - Fetch and cache repository index

### Infrastructure
- **pkg/gcs/gcs.go** (53 lines) - GCS client abstraction
  - `NewClient()` - Creates authenticated GCS client
  - `Object()` - Returns object handle for gs:// paths

### Configuration
- **plugin.yaml** - Helm plugin manifest (name, version, command, protocols)
- **go.mod** - Go module dependencies
- **.goreleaser.yml** - Cross-platform build configuration

---

## Important Patterns & Conventions

### Authentication (Priority Order)
1. **OAuth Token**: If `GOOGLE_OAUTH_ACCESS_TOKEN` env var is set
2. **Service Account**: If `--service-account` flag provided with JSON key file path
3. **ADC (Application Default Credentials)**: Fallback default method

See: cmd/helm-gcs/cmd/root.go:15-60

### Concurrency Control
Uses **optimistic locking** via GCS object generation numbers to prevent concurrent index corruption:
```go
o = o.If(storage.Conditions{GenerationMatch: r.indexFileGeneration})
```
Returns `ErrIndexOutOfDate` (HTTP 412) if index was modified concurrently. Use `--retry` flag for automatic retries.

See: pkg/repo/repo.go:320-340

### Error Handling
- Uses `pkg/errors` for error wrapping with context
- Custom error: `ErrIndexOutOfDate` for concurrent update detection
- Panics on init failure for immediate feedback on auth/config issues

### Logging
- Controlled by `--debug` flag or `HELM_GCS_DEBUG=true` env var
- Uses logrus structured logging (INFO/DEBUG levels)
- Outputs to stderr

---

## Technology Stack

| Component | Package | Version | Purpose |
|-----------|---------|---------|---------|
| CLI Framework | github.com/spf13/cobra | v1.10.2 | Command structure |
| GCS Client | cloud.google.com/go/storage | v1.39.1 | GCS operations |
| Helm Integration | helm.sh/helm/v3 | v3.18.5 | Chart/index handling |
| Authentication | golang.org/x/oauth2 | v0.28.0 | OAuth2 tokens |
| Logging | github.com/sirupsen/logrus | v1.9.3 | Structured logging |
| Error Handling | github.com/pkg/errors | v0.9.1 | Error wrapping |
| YAML Processing | github.com/ghodss/yaml | v1.0.0 | YAML marshal/unmarshal |

---

## Common Development Tasks

### Building the Project
```bash
go build -o bin/helm-gcs ./cmd/helm-gcs
```

### Running Tests
```bash
go test -race -v ./...
```

### Installing Locally
```bash
# From project root
./scripts/install.sh
```

### Adding a New CLI Command
1. Create new file in `cmd/helm-gcs/cmd/` (e.g., `mycmd.go`)
2. Define command using Cobra pattern (see existing commands)
3. Register in `root.go` init() function
4. Implement command logic (use `gcsClient` for GCS operations)

### Modifying Repository Operations
- Edit `pkg/repo/repo.go`
- Use `r.gcs.Object()` for GCS operations
- Remember to handle `ErrIndexOutOfDate` for concurrent updates
- Update index file after chart modifications

---

## Build & Release Process

### Build Configuration
- **Tool**: GoReleaser (`.goreleaser.yml`)
- **Targets**: linux/darwin/windows on amd64/arm64
- **Static Binary**: CGO_ENABLED=0
- **Version Injection**: Uses ldflags to inject version/commit/date

### CI/CD Workflows
- **test.yml** (on PRs):
  - Trivy security scanning
  - golangci-lint
  - gofmt check
  - go test with race detector
  - go vet
  - gocyclo (max complexity: 19)
  - Build verification

- **release.yml** (on tags):
  - Security scanning
  - GoReleaser build & publish
  - GitHub releases

### Code Quality Standards
- **Format**: `gofmt -s` required
- **Max Complexity**: 19 (cyclomatic)
- **Race Detection**: All tests run with `-race`
- **Linting**: golangci-lint (5min timeout)

---

## Helm Integration

### Plugin Registration
Defined in `plugin.yaml`:
- Plugin name: "gcs"
- Protocol handler: "gs"
- Download wrapper: `scripts/pull.sh`

### How It Works
1. User adds repo: `helm repo add my-repo gs://my-bucket/charts`
2. Helm calls helm-gcs via plugin interface
3. Plugin reads/writes to GCS using index.yaml format
4. On `helm install`, Helm calls `scripts/pull.sh` for gs:// URLs

### Index File Structure
Uses standard Helm `index.yaml` format stored at `gs://bucket/path/index.yaml`

---

## Important Code Locations

### Adding New Flags
- Global flags: cmd/helm-gcs/cmd/root.go (rootCmd.PersistentFlags())
- Command flags: Respective command files (e.g., push.go)

### Modifying GCS Operations
- Client setup: pkg/gcs/gcs.go:13-50
- Object operations: Use `*storage.ObjectHandle` from GCS SDK

### Changing Repository Logic
- Chart upload: pkg/repo/repo.go:187 (PushChart)
- Chart removal: pkg/repo/repo.go:233 (RemoveChart)
- Index management: pkg/repo/repo.go:288 (uploadIndexFile)

### Authentication Changes
- Auth logic: cmd/helm-gcs/cmd/root.go:36-60
- Environment variables checked: GOOGLE_OAUTH_ACCESS_TOKEN, HELM_GCS_DEBUG

---

## Debugging Tips

### Enable Debug Logging
```bash
export HELM_GCS_DEBUG=true
helm gcs push mychart-0.1.0.tgz gs://my-bucket/charts
```
or use `--debug` flag

### Common Issues

**Problem**: "index.yaml is out of date"
**Solution**: Use `--retry` flag or manually resolve conflicts

**Problem**: Authentication failures
**Solution**: Check ADC setup (`gcloud auth application-default login`) or provide `--service-account`

**Problem**: Concurrent updates
**Solution**: Use `--retry` flag for automatic retry with exponential backoff

---

## Dependencies Management

- **renovate.json**: Automated dependency updates via Renovate bot
- Recent updates: Go toolchain 1.25.1, Helm v3.18.5, Cobra v1.10.2
- Check `go.mod` for current versions
- Run `go mod tidy` after dependency changes

---

## Testing Strategy

### Current Coverage
- Unit tests in respective `*_test.go` files
- Race detector enabled in CI
- Static analysis (go vet, golangci-lint)
- Security scanning (Trivy)

### Adding Tests
- Place tests alongside source files
- Use table-driven tests where appropriate
- Mock GCS operations if needed
- Run with `-race` flag

---

## Security Considerations

1. **Credentials**: Never commit service account keys
2. **Public Buckets**: Use `--public` flag explicitly when intended
3. **Access Control**: Respect GCS IAM permissions
4. **Dependencies**: Automated scanning via Trivy in CI
5. **SARIF Reports**: Security findings uploaded to GitHub

---

## Quick Reference

### CLI Commands
```bash
helm gcs init gs://bucket/charts              # Initialize repository
helm gcs push chart.tgz gs://bucket/charts    # Push chart
helm gcs remove chart gs://bucket/charts      # Remove chart
helm gcs version                              # Show version info
```

### Important Environment Variables
- `GOOGLE_OAUTH_ACCESS_TOKEN` - OAuth token for auth
- `HELM_GCS_DEBUG` - Enable debug logging
- `HELM_REPOSITORY_CONFIG` - Helm repo config location
- `HELM_PLUGIN_DIR` - Plugin installation directory

### Global Flags
- `--service-account` - Path to service account JSON key
- `--debug` - Enable debug output

---

## Project Status

- **Maintenance**: Active (recent dependency updates)
- **Helm Version**: 3.x (0.3+ versions)
- **Go Version**: 1.24+ required, 1.25.1 toolchain
- **Platform Support**: Linux, macOS, Windows (amd64, arm64)

---

## Additional Resources

- **README.md** - User-facing documentation
- **CONTRIBUTING.md** - Contribution guidelines (if exists)
- **GitHub Issues** - Bug reports and feature requests
- **GitHub Actions** - CI/CD pipeline logs
