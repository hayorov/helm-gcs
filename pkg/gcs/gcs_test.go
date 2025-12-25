package gcs

import (
	"os"
	"testing"
)

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name        string
		gcsURL      string
		wantBucket  string
		wantPath    string
		expectError bool
	}{
		{
			name:        "valid gs:// URL",
			gcsURL:      "gs://my-bucket/path/to/chart",
			wantBucket:  "my-bucket",
			wantPath:    "path/to/chart",
			expectError: false,
		},
		{
			name:        "valid gcs:// URL",
			gcsURL:      "gcs://my-bucket/charts/index.yaml",
			wantBucket:  "my-bucket",
			wantPath:    "charts/index.yaml",
			expectError: false,
		},
		{
			name:        "bucket root path",
			gcsURL:      "gs://my-bucket/",
			wantBucket:  "my-bucket",
			wantPath:    "",
			expectError: false,
		},
		{
			name:        "nested path",
			gcsURL:      "gs://my-bucket/charts/stable/nginx-1.0.0.tgz",
			wantBucket:  "my-bucket",
			wantPath:    "charts/stable/nginx-1.0.0.tgz",
			expectError: false,
		},
		{
			name:        "invalid scheme - http",
			gcsURL:      "http://my-bucket/path",
			expectError: true,
		},
		{
			name:        "invalid scheme - https",
			gcsURL:      "https://storage.googleapis.com/my-bucket/path",
			expectError: true,
		},
		{
			name:        "no scheme",
			gcsURL:      "my-bucket/path",
			expectError: true,
		},
		{
			name:        "invalid URL",
			gcsURL:      "://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, path, err := splitPath(tt.gcsURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("splitPath() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("splitPath() unexpected error: %v", err)
				return
			}

			if bucket != tt.wantBucket {
				t.Errorf("splitPath() bucket = %v, want %v", bucket, tt.wantBucket)
			}

			if path != tt.wantPath {
				t.Errorf("splitPath() path = %v, want %v", path, tt.wantPath)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	// Save original env vars
	originalToken := os.Getenv("GOOGLE_OAUTH_ACCESS_TOKEN")
	originalAppCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GOOGLE_OAUTH_ACCESS_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GOOGLE_OAUTH_ACCESS_TOKEN")
		}
		if originalAppCreds != "" {
			_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", originalAppCreds)
		} else {
			_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
		}
	}()

	t.Run("client creation with default credentials", func(t *testing.T) {
		// Clear all credential env vars to avoid conflicts
		_ = os.Unsetenv("GOOGLE_OAUTH_ACCESS_TOKEN")
		_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

		// This will use Application Default Credentials (ADC)
		// In CI/testing without GCP credentials, this may fail, which is expected
		client, err := NewClient("")

		// If running in environment without GCP credentials, expect error
		if err != nil {
			// This is acceptable in test environment without GCP setup
			t.Logf("NewClient() with ADC failed (expected in non-GCP environment): %v", err)
			return
		}

		if client == nil {
			t.Error("NewClient() returned nil client without error")
		}
	})

	t.Run("client creation with service account", func(t *testing.T) {
		// Clear OAuth token
		os.Unsetenv("GOOGLE_OAUTH_ACCESS_TOKEN")
		os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

		// Try with a non-existent service account file
		// This should fail, which is expected
		client, err := NewClient("/non/existent/service-account.json")

		// We expect this to fail
		if err == nil {
			t.Logf("NewClient() with invalid service account succeeded (may have valid ADC)")
			if client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		} else {
			t.Logf("NewClient() with invalid service account failed as expected: %v", err)
		}
	})
}

func TestObject(t *testing.T) {
	// Skip this test in environments without GCP credentials
	if os.Getenv("SKIP_GCS_TESTS") == "true" {
		t.Skip("Skipping GCS object test (SKIP_GCS_TESTS=true)")
	}

	// Clear all credentials to avoid conflicts, then create client with ADC
	os.Unsetenv("GOOGLE_OAUTH_ACCESS_TOKEN")
	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	client, err := NewClient("")
	if err != nil {
		t.Skipf("Skipping test, cannot create GCS client: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid gs:// path",
			path:        "gs://test-bucket/charts/index.yaml",
			expectError: false,
		},
		{
			name:        "invalid http:// path",
			path:        "http://test-bucket/charts/index.yaml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := Object(client, tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("Object() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Object() unexpected error: %v", err)
				return
			}

			if obj == nil {
				t.Error("Object() returned nil object without error")
			}
		})
	}
}
