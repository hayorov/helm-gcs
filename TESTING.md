# Testing Guide for helm-gcs

This document describes the testing infrastructure and how to run tests for the helm-gcs project.

## Table of Contents

- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
- [Test Coverage](#test-coverage)
- [CI/CD Integration](#cicd-integration)
- [Writing New Tests](#writing-new-tests)

## Test Structure

The project uses a multi-tier testing approach:

```
helm-gcs/
├── cmd/
│   ├── helm-gcs/cmd/             # CLI plugin commands
│   └── helm-gcs-getter/
│       ├── main.go
│       └── main_test.go          # Getter binary tests
├── pkg/
│   ├── gcs/
│   │   ├── gcs.go
│   │   └── gcs_test.go           # Unit tests for GCS client
│   └── repo/
│       ├── repo.go
│       └── repo_test.go          # Unit tests for repository operations
├── integration/
│   └── integration_test.go       # Integration tests with real GCS bucket
├── testdata/
│   └── charts/
│       └── test-chart/           # Sample Helm chart for testing
└── Makefile                      # Test automation
```

### Test Categories

1. **Unit Tests** (`pkg/*/test.go`)
   - Test individual functions and methods
   - No external dependencies (GCS, network)
   - Fast execution
   - Run on every PR

2. **Integration Tests** (`integration/*_test.go`)
   - Test against real GCS buckets
   - Require GCP credentials
   - Longer execution time
   - Run manually or on schedule

## Running Tests

### Prerequisites

- Go 1.25 or later
- (For integration tests) GCS bucket and GCP credentials

### Quick Start - Local Development

**First time setup:**

```bash
# 1. Create .env file from template
make setup

# 2. Edit .env and configure your GCS bucket and credentials
vim .env  # or use your favorite editor

# 3. Verify setup
make help
# Should show: ✓ .env file found
#             ✓ GCS_TEST_BUCKET: gs://your-bucket/test-path
```

**Running tests:**

```bash
# Run all unit tests (reads .env automatically)
make test

# Run unit tests with coverage
make test-coverage

# Run integration tests (uses GCS_TEST_BUCKET from .env)
make test-integration

# Run all tests
make test-all
```

### Manual Configuration (without .env)

If you prefer not to use `.env`, set environment variables manually:

```bash
# Set GCS bucket for integration tests
export GCS_TEST_BUCKET=gs://your-bucket/test-path

# Set GCP credentials (choose one method)
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
# OR
export GOOGLE_OAUTH_ACCESS_TOKEN=$(gcloud auth print-access-token)
# OR use gcloud ADC
gcloud auth application-default login

# Run tests
make test-integration
```

### Using Go Commands Directly

```bash
# Unit tests only
go test -v -race ./pkg/...

# Specific package
go test -v -race ./pkg/gcs

# With coverage
go test -v -race -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out

# Integration tests
export GCS_TEST_BUCKET=gs://your-bucket/test-path
go test -v -race -tags=integration ./integration/...
```

## Unit Tests

Unit tests are located alongside the source code in `*_test.go` files.

### pkg/gcs Tests

Tests for GCS client functionality:

- `TestSplitPath`: URL parsing for gs:// and gcs:// schemes
- `TestNewClient`: Client creation with different auth methods
- `TestObject`: Object handle creation

**Run GCS tests:**
```bash
go test -v ./pkg/gcs
```

### pkg/repo Tests

Tests for repository operations:

- `TestResolveReference`: Path resolution
- `TestGetURL`: Public/private URL generation
- `TestLogger`: Logger configuration
- `TestEnvOr`: Environment variable handling
- `TestNew`: Repository initialization

**Run repo tests:**
```bash
go test -v ./pkg/repo
```

### Skipping Tests

Some tests are automatically skipped in environments without GCP credentials:

```bash
# Skip GCS-dependent tests
export SKIP_GCS_TESTS=true
go test -v ./pkg/...
```

## Integration Tests

Integration tests require a real GCS bucket and valid GCP credentials.

### Setup

#### Option 1: Using .env File (Recommended)

1. **Create .env file:**
   ```bash
   make setup
   ```

2. **Create a GCS bucket:**
   ```bash
   export PROJECT_ID=your-gcp-project-id
   gcloud config set project $PROJECT_ID

   # Create bucket
   gsutil mb -p $PROJECT_ID -l us-central1 gs://helm-gcs-test-bucket
   ```

3. **Set up authentication** (choose one method):

   **Method A: Service Account (Recommended)**
   ```bash
   # Create service account
   gcloud iam service-accounts create helm-gcs-dev \
       --display-name="Helm GCS Local Development"

   # Grant Storage Admin role
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member="serviceAccount:helm-gcs-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
       --role="roles/storage.admin"

   # Create and download key
   gcloud iam service-accounts keys create ~/helm-gcs-dev-key.json \
       --iam-account=helm-gcs-dev@${PROJECT_ID}.iam.gserviceaccount.com
   ```

   **Method B: Application Default Credentials (ADC)**
   ```bash
   gcloud auth application-default login
   ```

   **Method C: OAuth Token (Temporary)**
   ```bash
   # Token expires in 1 hour
   gcloud auth print-access-token
   ```

4. **Edit .env file:**
   ```bash
   # Open .env and add:
   GCS_TEST_BUCKET=gs://helm-gcs-test-bucket/integration-tests
   GOOGLE_APPLICATION_CREDENTIALS=/Users/yourname/helm-gcs-dev-key.json
   # OR leave GOOGLE_APPLICATION_CREDENTIALS empty to use ADC
   ```

5. **Verify setup:**
   ```bash
   make help
   # Should display:
   # Environment:
   #   ✓ .env file found
   #   ✓ GCS_TEST_BUCKET: gs://helm-gcs-test-bucket/integration-tests
   ```

#### Option 2: Manual Environment Variables

1. **Create a GCS bucket:**
   ```bash
   gsutil mb gs://your-test-bucket
   ```

2. **Set environment variables:**
   ```bash
   export GCS_TEST_BUCKET=gs://your-test-bucket/helm-gcs-integration-tests

   # Choose one authentication method:
   # Service Account:
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
   # OR ADC:
   gcloud auth application-default login
   # OR OAuth Token:
   export GOOGLE_OAUTH_ACCESS_TOKEN=$(gcloud auth print-access-token)
   ```

### Running Integration Tests

```bash
# Using Make
make test-integration GCS_TEST_BUCKET=gs://your-bucket/test-path

# Using go test
GCS_TEST_BUCKET=gs://your-bucket/test-path go test -v -race -tags=integration ./integration/...
```

### Integration Test Coverage

The integration test suite covers:

- **Repository Creation**: Creating a new repository on GCS
- **Push Chart**: Uploading Helm charts to the repository
- **Remove Chart**: Deleting charts from the repository
- **Concurrent Updates**: Testing optimistic locking with retry mechanism
- **Index Management**: Verifying index.yaml updates

### Cleanup

Integration tests automatically clean up resources after each test. If cleanup fails, you can manually remove test data:

```bash
gsutil -m rm -r gs://your-bucket/test-path/**
```

## Test Coverage

### Generating Coverage Reports

```bash
# Generate coverage report
make test-coverage

# View in browser
open coverage/coverage.html

# View summary
go tool cover -func=coverage/coverage.out
```

### Coverage Goals

- **Unit Tests**: Target 80%+ coverage for core functionality
- **Integration Tests**: Cover happy path and error scenarios
- **CI/CD**: Coverage reports uploaded to Codecov on PRs

## CI/CD Integration

### GitHub Actions Workflows

#### 1. Unit Tests (`test.yml`)

Runs on every PR:
- Code formatting check
- go vet static analysis
- golangci-lint
- Unit tests with race detector
- Code coverage upload to Codecov
- Binary build verification

**Trigger:**
```bash
git push origin feature-branch
# Opens PR -> tests run automatically
```

#### 2. Integration Tests (`integration-test.yml`)

Runs on:
- Manual trigger (workflow_dispatch)
- Weekly schedule (Sundays at 2 AM UTC)

**Manual trigger:**
1. Go to Actions tab in GitHub
2. Select "integration-tests" workflow
3. Click "Run workflow"
4. Enter GCS test bucket path

**Required secrets:**
- `GCS_TEST_CREDENTIALS`: Service account JSON key
- `GCS_TEST_BUCKET` (variable): Default test bucket path

### Setting Up CI/CD

To enable integration tests in CI:

1. **Create a GCP service account:**
   ```bash
   gcloud iam service-accounts create helm-gcs-tests \
     --display-name="Helm GCS Integration Tests"

   # Grant storage permissions
   gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
     --member="serviceAccount:helm-gcs-tests@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/storage.admin"

   # Create key
   gcloud iam service-accounts keys create gcs-test-key.json \
     --iam-account=helm-gcs-tests@YOUR_PROJECT_ID.iam.gserviceaccount.com
   ```

2. **Add GitHub secrets:**
   - Go to repository Settings → Secrets and variables → Actions
   - Add `GCS_TEST_CREDENTIALS` with the contents of `gcs-test-key.json`
   - Add variable `GCS_TEST_BUCKET` with your test bucket path

## Writing New Tests

### Unit Test Example

```go
package repo

import "testing"

func TestNewFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "test",
            wantErr: false,
        },
        {
            name:    "invalid input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFunction(tt.input)

            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got none")
                }
                return
            }

            if err != nil {
                t.Errorf("unexpected error: %v", err)
            }

            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Test Example

```go
// +build integration

package integration

import (
    "testing"
    "os"
)

func TestIntegration_NewFeature(t *testing.T) {
    // Get test bucket from environment
    testBucket := os.Getenv("GCS_TEST_BUCKET")
    if testBucket == "" {
        t.Skip("GCS_TEST_BUCKET not set")
    }

    // Test implementation
    // ...
}
```

### Best Practices

1. **Table-Driven Tests**: Use for multiple test cases
2. **Descriptive Names**: Test names should describe the scenario
3. **Error Messages**: Provide clear failure messages
4. **Cleanup**: Always clean up resources in tests
5. **Parallel Tests**: Use `t.Parallel()` where appropriate
6. **Test Data**: Store fixtures in `testdata/`

## Troubleshooting

### Common Issues

**Problem: "cannot find package" errors**
```bash
# Solution: Install dependencies
go mod download
go mod tidy
```

**Problem: Integration tests fail with auth errors**
```bash
# Solution: Check GCP authentication
gcloud auth application-default login
gcloud auth application-default print-access-token
```

**Problem: Tests timeout**
```bash
# Solution: Increase timeout
go test -timeout 10m ./...
```

**Problem: Race detector warnings**
```bash
# Solution: Fix race conditions or use mutex
# Run without race detector temporarily (not recommended):
go test ./...
```

### Debug Mode

Enable debug logging in tests:

```bash
export HELM_GCS_DEBUG=true
go test -v ./pkg/repo
```

## Test Metrics

Current test statistics:

- **Unit Tests**: Covering core GCS and repository functions
- **Integration Tests**: 4 major scenarios
- **Code Coverage**: Targeting 80%+
- **Test Execution Time**:
  - Unit tests: < 5 seconds
  - Integration tests: 1-2 minutes

## Contributing

When adding new features:

1. Write unit tests for new functions
2. Add integration tests for end-to-end flows
3. Ensure all tests pass: `make test-all`
4. Check coverage: `make test-coverage`
5. Run linters: `make lint`

For more information, see [CONTRIBUTING.md](./CONTRIBUTING.md).
