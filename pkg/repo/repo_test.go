package repo

import (
	"net/url"
	"os"
	"testing"
)

func TestResolveReference(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "simple path join",
			base:    "gs://my-bucket/charts",
			path:    "index.yaml",
			want:    "gs://my-bucket/charts/index.yaml",
			wantErr: false,
		},
		{
			name:    "path with trailing slash",
			base:    "gs://my-bucket/charts/",
			path:    "index.yaml",
			want:    "gs://my-bucket/charts/index.yaml",
			wantErr: false,
		},
		{
			name:    "nested path",
			base:    "gs://my-bucket/charts",
			path:    "stable/nginx-1.0.0.tgz",
			want:    "gs://my-bucket/charts/stable/nginx-1.0.0.tgz",
			wantErr: false,
		},
		{
			name:    "bucket root",
			base:    "gs://my-bucket",
			path:    "index.yaml",
			want:    "gs://my-bucket/index.yaml",
			wantErr: false,
		},
		{
			name:    "invalid base URL",
			base:    "://invalid",
			path:    "index.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveReference(tt.base, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("resolveReference() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("resolveReference() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("resolveReference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetURL(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		public    bool
		publicURL string
		want      string
		wantErr   bool
	}{
		{
			name:      "private URL unchanged",
			base:      "gs://my-bucket/charts",
			public:    false,
			publicURL: "",
			want:      "gs://my-bucket/charts",
			wantErr:   false,
		},
		{
			name:      "public URL with custom publicURL",
			base:      "gs://my-bucket/charts",
			public:    true,
			publicURL: "https://cdn.example.com/charts",
			want:      "https://cdn.example.com/charts",
			wantErr:   false,
		},
		{
			name:      "public URL with default googleapis",
			base:      "gs://my-bucket/charts",
			public:    true,
			publicURL: "",
			want:      "https://storage.googleapis.com/my-bucket//charts",
			wantErr:   false,
		},
		{
			name:      "public URL with nested path",
			base:      "gs://my-bucket/helm/stable",
			public:    true,
			publicURL: "",
			want:      "https://storage.googleapis.com/my-bucket//helm/stable",
			wantErr:   false,
		},
		{
			name:      "invalid base URL",
			base:      "://invalid",
			public:    false,
			publicURL: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getURL(tt.base, tt.public, tt.publicURL)

			if tt.wantErr {
				if err == nil {
					t.Error("getURL() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("getURL() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger(t *testing.T) {
	// Save original env var
	originalDebug := os.Getenv("HELM_GCS_DEBUG")
	defer func() {
		if originalDebug != "" {
			os.Setenv("HELM_GCS_DEBUG", originalDebug)
		} else {
			os.Unsetenv("HELM_GCS_DEBUG")
		}
	}()

	t.Run("logger with debug disabled", func(t *testing.T) {
		Debug = false
		os.Unsetenv("HELM_GCS_DEBUG")

		log := logger()
		if log == nil {
			t.Error("logger() returned nil")
		}
	})

	t.Run("logger with debug enabled via flag", func(t *testing.T) {
		Debug = true
		os.Unsetenv("HELM_GCS_DEBUG")

		log := logger()
		if log == nil {
			t.Error("logger() returned nil")
		}
	})

	t.Run("logger with debug enabled via env var", func(t *testing.T) {
		Debug = false
		os.Setenv("HELM_GCS_DEBUG", "true")

		log := logger()
		if log == nil {
			t.Error("logger() returned nil")
		}
	})

	// Reset Debug flag
	Debug = false
}

func TestEnvOr(t *testing.T) {
	tests := []struct {
		name     string
		envName  string
		envValue string
		setEnv   bool
		def      string
		want     string
	}{
		{
			name:     "env var set",
			envName:  "TEST_VAR_SET",
			envValue: "custom-value",
			setEnv:   true,
			def:      "default-value",
			want:     "custom-value",
		},
		{
			name:    "env var not set",
			envName: "TEST_VAR_UNSET",
			setEnv:  false,
			def:     "default-value",
			want:    "default-value",
		},
		{
			name:     "env var set to empty string",
			envName:  "TEST_VAR_EMPTY",
			envValue: "",
			setEnv:   true,
			def:      "default-value",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.envName, tt.envValue)
				defer os.Unsetenv(tt.envName)
			} else {
				os.Unsetenv(tt.envName)
			}

			got := envOr(tt.envName, tt.def)
			if got != tt.want {
				t.Errorf("envOr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	// This test requires a mock GCS client, so we'll test the basic flow
	// without actual GCS operations

	t.Run("create repo with valid path", func(t *testing.T) {
		// We can't create a real GCS client without credentials,
		// so this test validates the path handling
		path := "gs://test-bucket/charts"
		expectedIndexURL := "gs://test-bucket/charts/index.yaml"

		// Parse what the expected URL should be
		_, err := url.Parse(expectedIndexURL)
		if err != nil {
			t.Errorf("Expected index URL is invalid: %v", err)
		}

		// Test that resolveReference works correctly
		indexURL, err := resolveReference(path, "index.yaml")
		if err != nil {
			t.Errorf("resolveReference() error: %v", err)
		}

		if indexURL != expectedIndexURL {
			t.Errorf("resolveReference() = %v, want %v", indexURL, expectedIndexURL)
		}
	})

	t.Run("create repo with invalid path", func(t *testing.T) {
		path := "://invalid-url"

		_, err := resolveReference(path, "index.yaml")
		if err == nil {
			t.Error("Expected error for invalid URL, got none")
		}
	})
}

func TestErrIndexOutOfDate(t *testing.T) {
	// Verify that the error is properly defined
	if ErrIndexOutOfDate == nil {
		t.Error("ErrIndexOutOfDate should not be nil")
	}

	expectedMsg := "index is out-of-date"
	if ErrIndexOutOfDate.Error() != expectedMsg {
		t.Errorf("ErrIndexOutOfDate.Error() = %v, want %v", ErrIndexOutOfDate.Error(), expectedMsg)
	}
}
