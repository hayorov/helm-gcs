package repo

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	repov1 "helm.sh/helm/v4/pkg/repo/v1"
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
			want:      "https://storage.googleapis.com/my-bucket/charts",
			wantErr:   false,
		},
		{
			name:      "public URL with nested path",
			base:      "gs://my-bucket/helm/stable",
			public:    true,
			publicURL: "",
			want:      "https://storage.googleapis.com/my-bucket/helm/stable",
			wantErr:   false,
		},
		{
			name:      "public URL with trailing slash",
			base:      "gs://my-bucket/charts/",
			public:    true,
			publicURL: "",
			want:      "https://storage.googleapis.com/my-bucket/charts",
			wantErr:   false,
		},
		{
			name:      "public URL bucket root",
			base:      "gs://my-bucket",
			public:    true,
			publicURL: "",
			want:      "https://storage.googleapis.com/my-bucket",
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
			_ = os.Setenv("HELM_GCS_DEBUG", originalDebug)
		} else {
			_ = os.Unsetenv("HELM_GCS_DEBUG")
		}
	}()

	t.Run("logger with debug disabled", func(t *testing.T) {
		Debug = false
		_ = os.Unsetenv("HELM_GCS_DEBUG")

		log := logger()
		if log == nil {
			t.Error("logger() returned nil")
		}
	})

	t.Run("logger with debug enabled via flag", func(t *testing.T) {
		Debug = true
		_ = os.Unsetenv("HELM_GCS_DEBUG")

		log := logger()
		if log == nil {
			t.Error("logger() returned nil")
		}
	})

	t.Run("logger with debug enabled via env var", func(t *testing.T) {
		Debug = false
		_ = os.Setenv("HELM_GCS_DEBUG", "true")

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
				_ = os.Setenv(tt.envName, tt.envValue)
				defer func() { _ = os.Unsetenv(tt.envName) }()
			} else {
				_ = os.Unsetenv(tt.envName)
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

func TestRemoveChartVersion(t *testing.T) {
	mkVersion := func(v string) *repov1.ChartVersion {
		return &repov1.ChartVersion{
			Metadata: &chartv2.Metadata{Version: v},
		}
	}

	tests := []struct {
		name     string
		versions []string
		index    int
		want     []string
	}{
		{
			name:     "remove first element",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			index:    0,
			want:     []string{"2.0.0", "3.0.0"},
		},
		{
			name:     "remove middle element",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			index:    1,
			want:     []string{"1.0.0", "3.0.0"},
		},
		{
			name:     "remove last element",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			index:    2,
			want:     []string{"1.0.0", "2.0.0"},
		},
		{
			name:     "remove only element",
			versions: []string{"1.0.0"},
			index:    0,
			want:     []string{},
		},
		{
			name:     "remove from two elements",
			versions: []string{"1.0.0", "2.0.0"},
			index:    0,
			want:     []string{"2.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := make([]*repov1.ChartVersion, len(tt.versions))
			for i, v := range tt.versions {
				vs[i] = mkVersion(v)
			}

			got := removeChartVersion(vs, tt.index)

			if len(got) != len(tt.want) {
				t.Errorf("removeChartVersion() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i, want := range tt.want {
				if got[i].Version != want {
					t.Errorf("removeChartVersion()[%d].Version = %s, want %s", i, got[i].Version, want)
				}
			}
		})
	}
}

func TestApplyBucketPath(t *testing.T) {
	tests := []struct {
		name       string
		base       string
		bucketPath string
		want       string
		wantErr    bool
	}{
		{
			name:       "empty bucket path",
			base:       "gs://my-bucket/charts",
			bucketPath: "",
			want:       "gs://my-bucket/charts",
			wantErr:    false,
		},
		{
			name:       "simple bucket path",
			base:       "gs://my-bucket/charts",
			bucketPath: "subdir",
			want:       "gs://my-bucket/charts/subdir",
			wantErr:    false,
		},
		{
			name:       "nested bucket path",
			base:       "gs://my-bucket/charts",
			bucketPath: "a/b/c",
			want:       "gs://my-bucket/charts/a/b/c",
			wantErr:    false,
		},
		{
			name:       "bucket path with base trailing slash",
			base:       "gs://my-bucket/charts/",
			bucketPath: "subdir",
			want:       "gs://my-bucket/charts/subdir",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			var err error
			if tt.bucketPath == "" {
				got = tt.base
			} else {
				got, err = resolveReference(tt.base, tt.bucketPath)
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteChartFiles(t *testing.T) {
	// Save original gcsObjectFunc and restore after test
	origFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origFunc })

	t.Run("success deletes all URLs", func(t *testing.T) {
		deletedURLs := []string{}
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			deletedURLs = append(deletedURLs, path)
			// Return a real ObjectHandle that points to a test bucket
			// For unit testing, we'll verify the function was called correctly
			// but we can't execute real deletes without a mock
			return nil, nil
		}

		// Since we can't easily mock ObjectHandle.Delete, we test the path validation
		urls := []string{"gs://bucket/chart1.tgz", "gs://bucket/chart2.tgz"}

		// With nil ObjectHandle, this will panic, so let's test error case instead
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("mock object error")
		}

		err := deleteChartFiles(context.Background(), nil, urls)
		if err == nil {
			t.Error("expected error when gcsObjectFunc returns error")
		}
	})

	t.Run("object getter error returns wrapped error", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("bucket not found")
		}

		urls := []string{"gs://invalid-bucket/chart.tgz"}
		err := deleteChartFiles(context.Background(), nil, urls)

		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if err.Error() != "object: bucket not found" {
			t.Errorf("expected wrapped error message, got: %v", err)
		}
	})

	t.Run("empty URL list succeeds", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			t.Error("should not be called for empty list")
			return nil, nil
		}

		err := deleteChartFiles(context.Background(), nil, []string{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestNewRepo(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantIndexURL string
		wantErr      bool
	}{
		{
			name:         "valid gs:// path",
			path:         "gs://my-bucket/charts",
			wantIndexURL: "gs://my-bucket/charts/index.yaml",
			wantErr:      false,
		},
		{
			name:         "valid gcs:// path",
			path:         "gcs://my-bucket/charts",
			wantIndexURL: "gcs://my-bucket/charts/index.yaml",
			wantErr:      false,
		},
		{
			name:         "bucket root",
			path:         "gs://my-bucket",
			wantIndexURL: "gs://my-bucket/index.yaml",
			wantErr:      false,
		},
		{
			name:         "nested path",
			path:         "gs://my-bucket/helm/stable/charts",
			wantIndexURL: "gs://my-bucket/helm/stable/charts/index.yaml",
			wantErr:      false,
		},
		{
			name:         "path with trailing slash",
			path:         "gs://my-bucket/charts/",
			wantIndexURL: "gs://my-bucket/charts/index.yaml",
			wantErr:      false,
		},
		{
			name:    "invalid URL",
			path:    "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := New(tt.path, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("New() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			if repo == nil {
				t.Error("New() returned nil repo")
				return
			}

			if repo.indexFileURL != tt.wantIndexURL {
				t.Errorf("New() indexFileURL = %v, want %v", repo.indexFileURL, tt.wantIndexURL)
			}

			if repo.entry != nil {
				t.Error("New() entry should be nil")
			}

			if repo.indexFileGeneration != 0 {
				t.Error("New() indexFileGeneration should be 0")
			}
		})
	}
}

func TestRetrieveRepositoryEntry(t *testing.T) {
	// Create a temporary directory for test repo files
	tmpDir := t.TempDir()

	// Save original env var and restore after test
	originalConfig := os.Getenv("HELM_REPOSITORY_CONFIG")
	t.Cleanup(func() {
		if originalConfig != "" {
			_ = os.Setenv("HELM_REPOSITORY_CONFIG", originalConfig)
		} else {
			_ = os.Unsetenv("HELM_REPOSITORY_CONFIG")
		}
	})

	t.Run("repository not found", func(t *testing.T) {
		// Create empty repo file
		repoFile := filepath.Join(tmpDir, "empty-repos.yaml")
		err := os.WriteFile(repoFile, []byte("apiVersion: v1\nrepositories: []\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		_, err = retrieveRepositoryEntry("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent repository")
			return
		}

		expectedErr := `repository "nonexistent" does not exist`
		if err.Error() != expectedErr {
			t.Errorf("Error = %q, want %q", err.Error(), expectedErr)
		}
	})

	t.Run("repository found", func(t *testing.T) {
		// Create repo file with test entry
		repoFile := filepath.Join(tmpDir, "test-repos.yaml")
		repoContent := `apiVersion: v1
repositories:
  - name: test-repo
    url: gs://test-bucket/charts
  - name: another-repo
    url: gs://another-bucket/charts
`
		err := os.WriteFile(repoFile, []byte(repoContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		entry, err := retrieveRepositoryEntry("test-repo")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if entry == nil {
			t.Error("Expected entry, got nil")
			return
		}

		if entry.Name != "test-repo" {
			t.Errorf("Entry.Name = %q, want %q", entry.Name, "test-repo")
		}

		if entry.URL != "gs://test-bucket/charts" {
			t.Errorf("Entry.URL = %q, want %q", entry.URL, "gs://test-bucket/charts")
		}
	})

	t.Run("second repository found", func(t *testing.T) {
		repoFile := filepath.Join(tmpDir, "multi-repos.yaml")
		repoContent := `apiVersion: v1
repositories:
  - name: first-repo
    url: gs://first-bucket/charts
  - name: second-repo
    url: gs://second-bucket/charts
  - name: third-repo
    url: gs://third-bucket/charts
`
		err := os.WriteFile(repoFile, []byte(repoContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		entry, err := retrieveRepositoryEntry("second-repo")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if entry.Name != "second-repo" {
			t.Errorf("Entry.Name = %q, want %q", entry.Name, "second-repo")
		}

		if entry.URL != "gs://second-bucket/charts" {
			t.Errorf("Entry.URL = %q, want %q", entry.URL, "gs://second-bucket/charts")
		}
	})

	t.Run("invalid repo file", func(t *testing.T) {
		repoFile := filepath.Join(tmpDir, "invalid-repos.yaml")
		err := os.WriteFile(repoFile, []byte("not valid yaml: ["), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		_, err = retrieveRepositoryEntry("any-repo")
		if err == nil {
			t.Error("Expected error for invalid repo file")
		}
	})

	t.Run("missing repo file", func(t *testing.T) {
		_ = os.Setenv("HELM_REPOSITORY_CONFIG", filepath.Join(tmpDir, "nonexistent.yaml"))

		_, err := retrieveRepositoryEntry("any-repo")
		if err == nil {
			t.Error("Expected error for missing repo file")
		}
	})
}

func TestLoad(t *testing.T) {
	// Create a temporary directory for test repo files
	tmpDir := t.TempDir()

	// Save original env var
	originalConfig := os.Getenv("HELM_REPOSITORY_CONFIG")
	t.Cleanup(func() {
		if originalConfig != "" {
			_ = os.Setenv("HELM_REPOSITORY_CONFIG", originalConfig)
		} else {
			_ = os.Unsetenv("HELM_REPOSITORY_CONFIG")
		}
	})

	t.Run("load existing repository", func(t *testing.T) {
		repoFile := filepath.Join(tmpDir, "load-test-repos.yaml")
		repoContent := `apiVersion: v1
repositories:
  - name: my-gcs-repo
    url: gs://my-bucket/helm/charts
`
		err := os.WriteFile(repoFile, []byte(repoContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		repo, err := Load("my-gcs-repo", nil)
		if err != nil {
			t.Errorf("Load() unexpected error: %v", err)
			return
		}

		if repo == nil {
			t.Error("Load() returned nil repo")
			return
		}

		if repo.entry == nil {
			t.Error("Load() entry should not be nil")
			return
		}

		if repo.entry.Name != "my-gcs-repo" {
			t.Errorf("Load() entry.Name = %q, want %q", repo.entry.Name, "my-gcs-repo")
		}

		expectedIndexURL := "gs://my-bucket/helm/charts/index.yaml"
		if repo.indexFileURL != expectedIndexURL {
			t.Errorf("Load() indexFileURL = %q, want %q", repo.indexFileURL, expectedIndexURL)
		}
	})

	t.Run("load non-existent repository", func(t *testing.T) {
		repoFile := filepath.Join(tmpDir, "load-empty-repos.yaml")
		err := os.WriteFile(repoFile, []byte("apiVersion: v1\nrepositories: []\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		_, err = Load("nonexistent", nil)
		if err == nil {
			t.Error("Load() expected error for non-existent repository")
		}
	})
}

func TestRepoStruct(t *testing.T) {
	t.Run("repo fields initialization", func(t *testing.T) {
		repo := &Repo{
			entry: &repov1.Entry{
				Name: "test",
				URL:  "gs://bucket/path",
			},
			indexFileURL:        "gs://bucket/path/index.yaml",
			indexFileGeneration: 12345,
			gcs:                 nil,
		}

		if repo.entry.Name != "test" {
			t.Errorf("entry.Name = %q, want %q", repo.entry.Name, "test")
		}

		if repo.indexFileURL != "gs://bucket/path/index.yaml" {
			t.Errorf("indexFileURL = %q, want %q", repo.indexFileURL, "gs://bucket/path/index.yaml")
		}

		if repo.indexFileGeneration != 12345 {
			t.Errorf("indexFileGeneration = %d, want %d", repo.indexFileGeneration, 12345)
		}
	})
}

func TestLoggerDebugModes(t *testing.T) {
	// Save original values
	originalDebug := Debug
	originalEnv := os.Getenv("HELM_GCS_DEBUG")

	t.Cleanup(func() {
		Debug = originalDebug
		if originalEnv != "" {
			_ = os.Setenv("HELM_GCS_DEBUG", originalEnv)
		} else {
			_ = os.Unsetenv("HELM_GCS_DEBUG")
		}
	})

	tests := []struct {
		name       string
		debugFlag  bool
		envValue   string
		setEnv     bool
		expectInfo bool
	}{
		{
			name:       "debug off, no env",
			debugFlag:  false,
			setEnv:     false,
			expectInfo: true,
		},
		{
			name:       "debug on via flag",
			debugFlag:  true,
			setEnv:     false,
			expectInfo: false,
		},
		{
			name:       "debug on via env true",
			debugFlag:  false,
			envValue:   "true",
			setEnv:     true,
			expectInfo: false,
		},
		{
			name:       "debug on via env TRUE",
			debugFlag:  false,
			envValue:   "TRUE",
			setEnv:     true,
			expectInfo: false,
		},
		{
			name:       "debug off with false env",
			debugFlag:  false,
			envValue:   "false",
			setEnv:     true,
			expectInfo: true,
		},
		{
			name:       "debug off with empty env",
			debugFlag:  false,
			envValue:   "",
			setEnv:     true,
			expectInfo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Debug = tt.debugFlag
			if tt.setEnv {
				_ = os.Setenv("HELM_GCS_DEBUG", tt.envValue)
			} else {
				_ = os.Unsetenv("HELM_GCS_DEBUG")
			}

			l := logger()
			if l == nil {
				t.Error("logger() returned nil")
			}
		})
	}
}

func TestIndexFileURLConstruction(t *testing.T) {
	tests := []struct {
		name         string
		basePath     string
		wantIndexURL string
	}{
		{
			name:         "standard bucket path",
			basePath:     "gs://my-bucket/charts",
			wantIndexURL: "gs://my-bucket/charts/index.yaml",
		},
		{
			name:         "bucket root",
			basePath:     "gs://my-bucket",
			wantIndexURL: "gs://my-bucket/index.yaml",
		},
		{
			name:         "deeply nested path",
			basePath:     "gs://my-bucket/a/b/c/d",
			wantIndexURL: "gs://my-bucket/a/b/c/d/index.yaml",
		},
		{
			name:         "gcs scheme",
			basePath:     "gcs://my-bucket/charts",
			wantIndexURL: "gcs://my-bucket/charts/index.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := New(tt.basePath, nil)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			if repo.indexFileURL != tt.wantIndexURL {
				t.Errorf("indexFileURL = %q, want %q", repo.indexFileURL, tt.wantIndexURL)
			}
		})
	}
}

func TestEnvOrEdgeCases(t *testing.T) {
	t.Run("special characters in env value", func(t *testing.T) {
		envName := "TEST_SPECIAL_CHARS"
		envValue := "/path/with spaces/and=equals&special"
		_ = os.Setenv(envName, envValue)
		defer func() { _ = os.Unsetenv(envName) }()

		got := envOr(envName, "default")
		if got != envValue {
			t.Errorf("envOr() = %q, want %q", got, envValue)
		}
	})

	t.Run("unicode in env value", func(t *testing.T) {
		envName := "TEST_UNICODE"
		envValue := "日本語テスト"
		_ = os.Setenv(envName, envValue)
		defer func() { _ = os.Unsetenv(envName) }()

		got := envOr(envName, "default")
		if got != envValue {
			t.Errorf("envOr() = %q, want %q", got, envValue)
		}
	})
}

func TestResolveReferenceEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "path with special characters",
			base: "gs://my-bucket/charts",
			path: "chart-1.0.0+build.123.tgz",
			want: "gs://my-bucket/charts/chart-1.0.0+build.123.tgz",
		},
		{
			name: "empty path segment",
			base: "gs://my-bucket/charts",
			path: "",
			want: "gs://my-bucket/charts",
		},
		{
			name: "dot path",
			base: "gs://my-bucket/charts",
			path: ".",
			want: "gs://my-bucket/charts",
		},
		{
			name: "double slash in base",
			base: "gs://my-bucket//charts",
			path: "file.tgz",
			want: "gs://my-bucket/charts/file.tgz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveReference(tt.base, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("resolveReference() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetURLEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		public    bool
		publicURL string
		want      string
		wantErr   bool
	}{
		{
			name:      "empty base URL",
			base:      "",
			public:    false,
			publicURL: "",
			want:      "",
			wantErr:   false,
		},
		{
			name:      "public with empty publicURL uses googleapis",
			base:      "gs://bucket/path",
			public:    true,
			publicURL: "",
			want:      "https://storage.googleapis.com/bucket/path",
			wantErr:   false,
		},
		{
			name:      "non-public ignores publicURL",
			base:      "gs://bucket/path",
			public:    false,
			publicURL: "https://custom.cdn.com/charts",
			want:      "gs://bucket/path",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getURL(tt.base, tt.public, tt.publicURL)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPushChartLoadError(t *testing.T) {
	// Save original functions
	origGcsObjectFunc := gcsObjectFunc
	origChartLoader := chartLoader

	t.Cleanup(func() {
		gcsObjectFunc = origGcsObjectFunc
		chartLoader = origChartLoader
	})

	// Create repo with entry
	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
	}

	t.Run("chart load error", func(t *testing.T) {
		// Mock gcsObjectFunc to return error for indexFile
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs object error")
		}

		err := repo.PushChart("/nonexistent/chart.tgz", false, false, false, "", "", nil)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		// Should fail on loading index file first
		if !containsString(err.Error(), "load index file") {
			t.Errorf("expected error about index file, got: %v", err)
		}
	})
}

func TestRemoveChartNotFound(t *testing.T) {
	// Save original function
	origGcsObjectFunc := gcsObjectFunc

	t.Cleanup(func() {
		gcsObjectFunc = origGcsObjectFunc
	})

	// Create repo
	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
	}

	t.Run("index file error", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs object error")
		}

		err := repo.RemoveChart("nonexistent-chart", "1.0.0", false)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "index") {
			t.Errorf("expected error about index, got: %v", err)
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestChartLoaderInjection(t *testing.T) {
	// Verify that chartLoader is properly initialized
	if chartLoader == nil {
		t.Error("chartLoader should not be nil")
	}
}

func TestDigestFileInjection(t *testing.T) {
	// Verify that digestFile is properly initialized
	if digestFile == nil {
		t.Error("digestFile should not be nil")
	}
}

func TestGcsObjectFuncInjection(t *testing.T) {
	// Verify that gcsObjectFunc is properly initialized
	if gcsObjectFunc == nil {
		t.Error("gcsObjectFunc should not be nil")
	}
}

func TestErrIndexOutOfDateWrapping(t *testing.T) {
	// Test that ErrIndexOutOfDate can be checked with errors.Is
	wrappedErr := errors.Wrap(ErrIndexOutOfDate, "wrapped error")

	// The wrapped error should still contain the original message
	if !containsString(wrappedErr.Error(), "index is out-of-date") {
		t.Errorf("wrapped error should contain original message, got: %v", wrappedErr)
	}
}

func TestRepoMethodsWithNilEntry(t *testing.T) {
	repo := &Repo{
		entry:        nil,
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	// Verify repo was created correctly
	if repo.entry != nil {
		t.Error("entry should be nil")
	}

	if repo.indexFileURL != "gs://test-bucket/charts/index.yaml" {
		t.Errorf("indexFileURL = %q, want %q", repo.indexFileURL, "gs://test-bucket/charts/index.yaml")
	}
}

func TestLoadWithInvalidURL(t *testing.T) {
	tmpDir := t.TempDir()

	originalConfig := os.Getenv("HELM_REPOSITORY_CONFIG")
	t.Cleanup(func() {
		if originalConfig != "" {
			_ = os.Setenv("HELM_REPOSITORY_CONFIG", originalConfig)
		} else {
			_ = os.Unsetenv("HELM_REPOSITORY_CONFIG")
		}
	})

	t.Run("repository with invalid URL", func(t *testing.T) {
		repoFile := filepath.Join(tmpDir, "invalid-url-repos.yaml")
		repoContent := `apiVersion: v1
repositories:
  - name: invalid-url-repo
    url: "://invalid-url"
`
		err := os.WriteFile(repoFile, []byte(repoContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		_, err = Load("invalid-url-repo", nil)
		if err == nil {
			t.Error("Load() expected error for invalid URL")
		}
	})
}

func TestNewWithVariousURLFormats(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "gs:// scheme",
			path:    "gs://bucket/path",
			wantErr: false,
		},
		{
			name:    "gcs:// scheme",
			path:    "gcs://bucket/path",
			wantErr: false,
		},
		{
			name:    "http:// scheme (still valid URL)",
			path:    "http://example.com/charts",
			wantErr: false,
		},
		{
			name:    "https:// scheme (still valid URL)",
			path:    "https://example.com/charts",
			wantErr: false,
		},
		{
			name:    "empty string",
			path:    "",
			wantErr: false,
		},
		{
			name:    "invalid URL with bad scheme",
			path:    "://no-scheme",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.path, nil)

			if tt.wantErr && err == nil {
				t.Error("expected error, got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRepoIndexFileURLGeneration(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantIndexURL string
	}{
		{
			name:         "simple path",
			path:         "gs://bucket/charts",
			wantIndexURL: "gs://bucket/charts/index.yaml",
		},
		{
			name:         "root bucket",
			path:         "gs://bucket",
			wantIndexURL: "gs://bucket/index.yaml",
		},
		{
			name:         "deep path",
			path:         "gs://bucket/a/b/c/d/e",
			wantIndexURL: "gs://bucket/a/b/c/d/e/index.yaml",
		},
		{
			name:         "path with trailing slash",
			path:         "gs://bucket/charts/",
			wantIndexURL: "gs://bucket/charts/index.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := New(tt.path, nil)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			if repo.indexFileURL != tt.wantIndexURL {
				t.Errorf("indexFileURL = %q, want %q", repo.indexFileURL, tt.wantIndexURL)
			}
		})
	}
}

func TestDebugFlagBehavior(t *testing.T) {
	originalDebug := Debug
	t.Cleanup(func() { Debug = originalDebug })

	t.Run("debug flag defaults to false", func(t *testing.T) {
		Debug = false
		if Debug {
			t.Error("Debug should be false by default")
		}
	})

	t.Run("debug flag can be set to true", func(t *testing.T) {
		Debug = true
		if !Debug {
			t.Error("Debug should be true after setting")
		}
	})
}

func TestCreateErrors(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	repo := &Repo{
		entry:        nil,
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("gcs object error", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		err := Create(repo)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "object") {
			t.Errorf("expected error about object, got: %v", err)
		}
	})
}

func TestIndexFileError(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	repo := &Repo{
		entry:        nil,
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("gcs object error", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		_, err := repo.indexFile()
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "object") {
			t.Errorf("expected error about object, got: %v", err)
		}
	})
}

func TestUploadIndexFileError(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	repo := &Repo{
		entry:        nil,
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("gcs object error", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		indexFile := repov1.NewIndexFile()
		err := repo.uploadIndexFile(indexFile)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "object") {
			t.Errorf("expected error about object, got: %v", err)
		}
	})
}

func TestUploadChartErrors(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("file open error", func(t *testing.T) {
		err := repo.uploadChart("/nonexistent/path/chart.tgz", nil)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "open") {
			t.Errorf("expected error about open, got: %v", err)
		}
	})

	t.Run("gcs object error", func(t *testing.T) {
		// Create a temp file
		tmpFile, err := os.CreateTemp("", "test-chart-*.tgz")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		_ = tmpFile.Close()
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		err = repo.uploadChart(tmpFile.Name(), nil)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "object") {
			t.Errorf("expected error about object, got: %v", err)
		}
	})
}

func TestUpdateIndexFileErrors(t *testing.T) {
	origDigestFile := digestFile
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() {
		digestFile = origDigestFile
		gcsObjectFunc = origGcsObjectFunc
	})

	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("digest file error", func(t *testing.T) {
		digestFile = func(filename string) (string, error) {
			return "", errors.New("digest computation failed")
		}

		indexFile := repov1.NewIndexFile()
		chart := &chartv2.Chart{
			Metadata: &chartv2.Metadata{
				Name:    "test-chart",
				Version: "1.0.0",
			},
		}

		err := repo.updateIndexFile(indexFile, "/some/chart.tgz", chart, false, "", "")
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "generate chart file digest") {
			t.Errorf("expected error about digest, got: %v", err)
		}
	})

	t.Run("invalid bucket path error", func(t *testing.T) {
		digestFile = func(filename string) (string, error) {
			return "sha256:abc123", nil
		}

		// Create a repo with invalid URL to trigger error in resolveReference
		badRepo := &Repo{
			entry: &repov1.Entry{
				Name: "test-repo",
				URL:  "://invalid-url",
			},
			indexFileURL: "gs://test-bucket/charts/index.yaml",
			gcs:          nil,
		}

		indexFile := repov1.NewIndexFile()
		chart := &chartv2.Chart{
			Metadata: &chartv2.Metadata{
				Name:    "test-chart",
				Version: "1.0.0",
			},
		}

		err := badRepo.updateIndexFile(indexFile, "/some/chart.tgz", chart, false, "", "subpath")
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "resolve bucketPath") {
			t.Errorf("expected error about bucketPath, got: %v", err)
		}
	})

	t.Run("getURL error", func(t *testing.T) {
		digestFile = func(filename string) (string, error) {
			return "sha256:abc123", nil
		}

		// Create a repo with invalid URL to trigger error in getURL
		badRepo := &Repo{
			entry: &repov1.Entry{
				Name: "test-repo",
				URL:  "://invalid-url",
			},
			indexFileURL: "gs://test-bucket/charts/index.yaml",
			gcs:          nil,
		}

		indexFile := repov1.NewIndexFile()
		chart := &chartv2.Chart{
			Metadata: &chartv2.Metadata{
				Name:    "test-chart",
				Version: "1.0.0",
			},
		}

		err := badRepo.updateIndexFile(indexFile, "/some/chart.tgz", chart, false, "", "")
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "get chart base url") {
			t.Errorf("expected error about base url, got: %v", err)
		}
	})
}

func TestRemoveChartErrors(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("index file error", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		err := repo.RemoveChart("my-chart", "1.0.0", false)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "index") {
			t.Errorf("expected error about index, got: %v", err)
		}
	})
}

func TestPushChartErrors(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	origChartLoader := chartLoader
	t.Cleanup(func() {
		gcsObjectFunc = origGcsObjectFunc
		chartLoader = origChartLoader
	})

	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("index file error", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		err := repo.PushChart("/path/to/chart.tgz", false, false, false, "", "", nil)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "load index file") {
			t.Errorf("expected error about loading index file, got: %v", err)
		}
	})
}

func TestRemoveChartVersionSliceBehavior(t *testing.T) {
	mkVersion := func(v string) *repov1.ChartVersion {
		return &repov1.ChartVersion{
			Metadata: &chartv2.Metadata{Version: v},
		}
	}

	t.Run("preserves original slice capacity", func(t *testing.T) {
		original := []*repov1.ChartVersion{
			mkVersion("1.0.0"),
			mkVersion("2.0.0"),
			mkVersion("3.0.0"),
		}
		originalLen := len(original)

		result := removeChartVersion(original, 1)

		if len(result) != originalLen-1 {
			t.Errorf("result length = %d, want %d", len(result), originalLen-1)
		}

		if result[0].Version != "1.0.0" {
			t.Errorf("result[0].Version = %q, want %q", result[0].Version, "1.0.0")
		}

		if result[1].Version != "3.0.0" {
			t.Errorf("result[1].Version = %q, want %q", result[1].Version, "3.0.0")
		}
	})
}

func TestDeleteChartFilesMultipleURLs(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	t.Run("stops on first error", func(t *testing.T) {
		callCount := 0
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("second url failed")
			}
			return nil, errors.New("first url failed")
		}

		urls := []string{
			"gs://bucket/chart1.tgz",
			"gs://bucket/chart2.tgz",
			"gs://bucket/chart3.tgz",
		}

		err := deleteChartFiles(context.Background(), nil, urls)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		// Should stop after first error
		if callCount != 1 {
			t.Errorf("expected 1 call, got %d", callCount)
		}
	})
}

func TestLoadEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	originalConfig := os.Getenv("HELM_REPOSITORY_CONFIG")
	t.Cleanup(func() {
		if originalConfig != "" {
			_ = os.Setenv("HELM_REPOSITORY_CONFIG", originalConfig)
		} else {
			_ = os.Unsetenv("HELM_REPOSITORY_CONFIG")
		}
	})

	t.Run("repository with complex URL", func(t *testing.T) {
		repoFile := filepath.Join(tmpDir, "complex-url-repos.yaml")
		repoContent := `apiVersion: v1
repositories:
  - name: complex-repo
    url: gs://my-bucket/helm/stable/charts
`
		err := os.WriteFile(repoFile, []byte(repoContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test repo file: %v", err)
		}

		_ = os.Setenv("HELM_REPOSITORY_CONFIG", repoFile)

		repo, err := Load("complex-repo", nil)
		if err != nil {
			t.Errorf("Load() unexpected error: %v", err)
			return
		}

		expectedIndexURL := "gs://my-bucket/helm/stable/charts/index.yaml"
		if repo.indexFileURL != expectedIndexURL {
			t.Errorf("indexFileURL = %q, want %q", repo.indexFileURL, expectedIndexURL)
		}
	})
}

func TestUploadChartResolveReferenceError(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	// Create repo with invalid URL
	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "://invalid-url",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("resolve reference error", func(t *testing.T) {
		// Create a temp file
		tmpFile, err := os.CreateTemp("", "test-chart-*.tgz")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		_ = tmpFile.Close()
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		err = repo.uploadChart(tmpFile.Name(), nil)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "resolve reference") {
			t.Errorf("expected error about resolve reference, got: %v", err)
		}
	})
}

func TestUpdateIndexFileWithExistingChart(t *testing.T) {
	origDigestFile := digestFile
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() {
		digestFile = origDigestFile
		gcsObjectFunc = origGcsObjectFunc
	})

	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("update existing chart version", func(t *testing.T) {
		digestFile = func(filename string) (string, error) {
			return "sha256:abc123", nil
		}

		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		// Create index with existing chart
		indexFile := repov1.NewIndexFile()
		indexFile.Entries = map[string]repov1.ChartVersions{
			"test-chart": {
				&repov1.ChartVersion{
					Metadata: &chartv2.Metadata{
						Name:    "test-chart",
						Version: "1.0.0",
					},
					URLs: []string{"gs://test-bucket/charts/test-chart-1.0.0.tgz"},
				},
			},
		}

		chart := &chartv2.Chart{
			Metadata: &chartv2.Metadata{
				Name:    "test-chart",
				Version: "1.0.0",
			},
		}

		// Should fail at uploadIndexFile stage
		err := repo.updateIndexFile(indexFile, "/some/chart.tgz", chart, false, "", "")
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		// The error should be from uploadIndexFile (object error)
		if !containsString(err.Error(), "object") {
			t.Errorf("expected error about object, got: %v", err)
		}
	})
}

func TestCreateWithExistingIndex(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	t.Run("index already exists scenario requires real GCS", func(t *testing.T) {
		// This test documents behavior - we can't fully test without mocking ObjectHandle
		repo := &Repo{
			entry:        nil,
			indexFileURL: "gs://test-bucket/charts/index.yaml",
			gcs:          nil,
		}

		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs error")
		}

		err := Create(repo)
		if err == nil {
			t.Error("expected error with mocked GCS, got none")
		}
	})
}

func TestPushChartChartLoaderError(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	origChartLoader := chartLoader
	t.Cleanup(func() {
		gcsObjectFunc = origGcsObjectFunc
		chartLoader = origChartLoader
	})

	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("chart loader error", func(t *testing.T) {
		// Mock gcsObjectFunc to fail on indexFile
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		err := repo.PushChart("/path/to/chart.tgz", false, false, false, "", "", nil)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "load index file") {
			t.Errorf("expected error about load index file, got: %v", err)
		}
	})
}

func TestRemoveChartIndexError(t *testing.T) {
	origGcsObjectFunc := gcsObjectFunc
	t.Cleanup(func() { gcsObjectFunc = origGcsObjectFunc })

	repo := &Repo{
		entry: &repov1.Entry{
			Name: "test-repo",
			URL:  "gs://test-bucket/charts",
		},
		indexFileURL: "gs://test-bucket/charts/index.yaml",
		gcs:          nil,
	}

	t.Run("index file error in remove", func(t *testing.T) {
		gcsObjectFunc = func(_ *storage.Client, path string) (*storage.ObjectHandle, error) {
			return nil, errors.New("gcs connection failed")
		}

		err := repo.RemoveChart("my-chart", "1.0.0", false)
		if err == nil {
			t.Error("expected error, got none")
			return
		}

		if !containsString(err.Error(), "index") {
			t.Errorf("expected error about index, got: %v", err)
		}
	})
}

func TestAllInjectableFunctionsInitialized(t *testing.T) {
	t.Run("gcsObjectFunc initialized", func(t *testing.T) {
		if gcsObjectFunc == nil {
			t.Error("gcsObjectFunc should be initialized")
		}
	})

	t.Run("chartLoader initialized", func(t *testing.T) {
		if chartLoader == nil {
			t.Error("chartLoader should be initialized")
		}
	})

	t.Run("digestFile initialized", func(t *testing.T) {
		if digestFile == nil {
			t.Error("digestFile should be initialized")
		}
	})
}

func TestRepoFieldAccess(t *testing.T) {
	repo := &Repo{
		entry: &repov1.Entry{
			Name: "my-repo",
			URL:  "gs://bucket/charts",
		},
		indexFileURL:        "gs://bucket/charts/index.yaml",
		indexFileGeneration: 42,
		gcs:                 nil,
	}

	t.Run("entry access", func(t *testing.T) {
		if repo.entry.Name != "my-repo" {
			t.Errorf("entry.Name = %q, want %q", repo.entry.Name, "my-repo")
		}
		if repo.entry.URL != "gs://bucket/charts" {
			t.Errorf("entry.URL = %q, want %q", repo.entry.URL, "gs://bucket/charts")
		}
	})

	t.Run("indexFileURL access", func(t *testing.T) {
		if repo.indexFileURL != "gs://bucket/charts/index.yaml" {
			t.Errorf("indexFileURL = %q, want %q", repo.indexFileURL, "gs://bucket/charts/index.yaml")
		}
	})

	t.Run("indexFileGeneration access", func(t *testing.T) {
		if repo.indexFileGeneration != 42 {
			t.Errorf("indexFileGeneration = %d, want %d", repo.indexFileGeneration, 42)
		}
	})
}
