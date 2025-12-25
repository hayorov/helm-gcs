# Development Guide

This guide helps you set up a local development environment for helm-gcs.

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/hayorov/helm-gcs.git
cd helm-gcs

# 2. Install dependencies
go mod download

# 3. Setup environment
make setup

# 4. Edit .env file with your GCS bucket and credentials
# See "Environment Configuration" section below

# 5. Run tests
make test

# 6. Build binary
make build

# 7. Test the binary
./bin/helm-gcs version
```

## Environment Configuration

### Using .env File (Recommended)

1. **Create .env file:**
   ```bash
   make setup
   ```

2. **Edit `.env`** and configure at minimum:
   ```bash
   # Required for integration tests
   GCS_TEST_BUCKET=gs://your-bucket/test-path

   # Authentication (choose one)
   GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
   # OR leave empty to use ADC (gcloud auth application-default login)
   ```

3. **Verify configuration:**
   ```bash
   make help
   # Should show ✓ for .env file and GCS_TEST_BUCKET
   ```

### Environment Variables Reference

See `.env.example` for full configuration options:

- `GCS_TEST_BUCKET` - GCS bucket for integration tests
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to service account JSON key
- `GOOGLE_OAUTH_ACCESS_TOKEN` - OAuth token (temporary)
- `HELM_GCS_DEBUG` - Enable debug logging
- `SKIP_GCS_TESTS` - Skip tests requiring GCS access

## Development Workflow

### Running Tests

```bash
# Unit tests only (no GCS required)
make test

# Integration tests (requires GCS bucket)
make test-integration

# All tests
make test-all

# With coverage report
make test-coverage
open coverage/coverage.html
```

### Code Quality

```bash
# Format code
make fmt

# Run static analysis
make vet

# Run linters
make lint

# Run all checks
make check
```

### Building

```bash
# Build for current platform
make build

# Binary will be at: ./bin/helm-gcs
./bin/helm-gcs version

# Test the plugin locally
helm plugin install .
helm gcs version
```

## Setting Up GCP for Development

### Option 1: Service Account (Recommended)

```bash
# Set your project
export PROJECT_ID=your-project-id
gcloud config set project $PROJECT_ID

# Create service account
gcloud iam service-accounts create helm-gcs-dev \
    --display-name="Helm GCS Development"

# Grant permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:helm-gcs-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/storage.admin"

# Create key file
gcloud iam service-accounts keys create ~/.gcp/helm-gcs-dev.json \
    --iam-account=helm-gcs-dev@${PROJECT_ID}.iam.gserviceaccount.com

# Add to .env
echo "GOOGLE_APPLICATION_CREDENTIALS=$HOME/.gcp/helm-gcs-dev.json" >> .env
```

### Option 2: Application Default Credentials (ADC)

```bash
# Login with your user account
gcloud auth application-default login

# No need to set GOOGLE_APPLICATION_CREDENTIALS in .env
# ADC will be used automatically
```

### Create Test Bucket

```bash
# Create bucket for testing
gsutil mb -p $PROJECT_ID -l us-central1 gs://helm-gcs-dev-tests

# Add to .env
echo "GCS_TEST_BUCKET=gs://helm-gcs-dev-tests/integration" >> .env

# Verify access
gsutil ls gs://helm-gcs-dev-tests
```

## Project Structure

```
helm-gcs/
├── cmd/helm-gcs/           # CLI application
│   ├── main.go            # Entry point
│   └── cmd/               # Cobra commands
├── pkg/                    # Core packages
│   ├── gcs/               # GCS client wrapper
│   └── repo/              # Repository operations
├── integration/            # Integration tests
├── testdata/              # Test fixtures
├── .env.example           # Environment template
├── Makefile               # Development automation
├── TESTING.md             # Testing guide
└── DEVELOPMENT.md         # This file
```

## Common Tasks

### Adding a New Command

1. Create new file in `cmd/helm-gcs/cmd/`:
   ```bash
   touch cmd/helm-gcs/cmd/mycommand.go
   ```

2. Define command using Cobra:
   ```go
   var myCmd = &cobra.Command{
       Use:   "mycommand",
       Short: "Description",
       Run: func(cmd *cobra.Command, args []string) {
           // Implementation
       },
   }

   func init() {
       rootCmd.AddCommand(myCmd)
   }
   ```

3. Add tests:
   ```bash
   touch cmd/helm-gcs/cmd/mycommand_test.go
   ```

### Adding Tests

1. **Unit tests** - add to `pkg/*_test.go`:
   ```go
   func TestNewFunction(t *testing.T) {
       tests := []struct {
           name    string
           input   string
           want    string
           wantErr bool
       }{
           // Test cases
       }
       // Implementation
   }
   ```

2. **Integration tests** - add to `integration/`:
   ```go
   // +build integration

   func TestIntegration_NewFeature(t *testing.T) {
       // Use real GCS bucket
   }
   ```

### Debugging

```bash
# Enable debug logging
export HELM_GCS_DEBUG=true
go test -v ./pkg/repo

# Or in .env
echo "HELM_GCS_DEBUG=true" >> .env

# Run with debugger (dlv)
dlv debug ./cmd/helm-gcs -- version
```

## Troubleshooting

### "Cannot find package" errors

```bash
go mod download
go mod tidy
```

### GCS authentication failures

```bash
# Check ADC
gcloud auth application-default print-access-token

# Verify service account
gcloud iam service-accounts list

# Test GCS access
gsutil ls gs://your-bucket
```

### Tests fail with timeout

```bash
# Increase timeout
go test -timeout 10m ./...
```

## Contributing

1. **Fork** the repository
2. **Create** a feature branch
3. **Make** your changes
4. **Add** tests
5. **Run** checks: `make check`
6. **Commit** with clear message
7. **Push** and create PR

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## IDE Setup

### VS Code

Recommended extensions:
- Go (golang.go)
- Go Test Explorer
- YAML

`.vscode/settings.json`:
```json
{
  "go.testEnvFile": "${workspaceFolder}/.env",
  "go.testFlags": ["-v"],
  "go.coverOnSave": true
}
```

### GoLand / IntelliJ IDEA

1. Settings → Go → Build Tags & Vendoring
2. Add custom build tags: `integration` (for integration tests)
3. Settings → Go → Environment
4. Add environment file: `.env`

## Resources

- [TESTING.md](./TESTING.md) - Comprehensive testing guide
- [.env.example](./.env.example) - Environment configuration template
- [Makefile](./Makefile) - All available commands
- [GitHub Workflows](./.github/workflows/README.md) - CI/CD setup guide

## Getting Help

- **Issues**: https://github.com/hayorov/helm-gcs/issues
- **Discussions**: https://github.com/hayorov/helm-gcs/discussions
- **Slack**: #helm-gcs on Kubernetes Slack
