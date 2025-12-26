//go:build integration
// +build integration

package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/joho/godotenv"
	"helm.sh/helm/v4/pkg/chart/loader"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	repo "helm.sh/helm/v4/pkg/repo/v1"

	"github.com/hayorov/helm-gcs/pkg/gcs"
)

// Integration tests require a real GCS bucket
// Set GCS_TEST_BUCKET environment variable to run these tests
// Example: GCS_TEST_BUCKET=gs://my-test-bucket/helm-gcs-tests go test -tags=integration ./integration

var (
	testBucket string
	gcsClient  *storage.Client
)

func TestMain(m *testing.M) {
	// Load environment variables from .env file (if exists)
	// Try loading from project root
	envPath := "../../.env"
	if err := godotenv.Load(envPath); err == nil {
		fmt.Println("✓ Loaded environment variables from .env file")
	} else {
		// .env file not found or error reading - that's okay, will use system env vars
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			fmt.Println("ℹ No .env file found, using system environment variables")
		}
	}

	// Check if integration tests should run
	testBucket = os.Getenv("GCS_TEST_BUCKET")
	if testBucket == "" {
		fmt.Println("Skipping integration tests: GCS_TEST_BUCKET not set")
		fmt.Println("To run integration tests:")
		fmt.Println("  1. Copy .env.example to .env and fill in GCS_TEST_BUCKET")
		fmt.Println("  2. Or set: export GCS_TEST_BUCKET=gs://your-bucket/test-path")
		os.Exit(0)
	}

	// Display authentication method
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("Integration Test Configuration")
	fmt.Println("========================================")
	fmt.Printf("GCS Bucket: %s\n", testBucket)

	// Check authentication method
	if oauthToken := os.Getenv("GOOGLE_OAUTH_ACCESS_TOKEN"); oauthToken != "" {
		fmt.Println("Auth Method: OAuth Access Token")
		fmt.Printf("Token: %s...%s (length: %d)\n", oauthToken[:10], oauthToken[len(oauthToken)-10:], len(oauthToken))
	} else if credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsFile != "" {
		fmt.Println("Auth Method: Service Account Key File")
		fmt.Printf("Credentials File: %s\n", credsFile)
		// Verify file exists
		if _, err := os.Stat(credsFile); os.IsNotExist(err) {
			fmt.Printf("⚠ WARNING: Credentials file does not exist: %s\n", credsFile)
		} else {
			fmt.Println("✓ Credentials file exists")
		}
	} else {
		fmt.Println("Auth Method: Application Default Credentials (ADC)")
		fmt.Println("ℹ Using gcloud auth application-default credentials")
	}

	if debug := os.Getenv("HELM_GCS_DEBUG"); debug == "true" || debug == "1" {
		fmt.Println("Debug Mode: ENABLED")
		Debug = true
	} else {
		fmt.Println("Debug Mode: disabled")
	}
	fmt.Println("========================================")
	fmt.Println()

	// Create GCS client
	var err error
	gcsClient, err = gcs.NewClient("")
	if err != nil {
		fmt.Printf("❌ Failed to create GCS client: %v\n", err)
		fmt.Println()
		fmt.Println("Troubleshooting:")
		fmt.Println("  1. Ensure GOOGLE_APPLICATION_CREDENTIALS points to a valid service account key")
		fmt.Println("  2. Or run: gcloud auth application-default login")
		fmt.Println("  3. Verify the service account has 'Storage Admin' or 'Storage Object Admin' role")
		os.Exit(1)
	}
	fmt.Println("✓ GCS client created successfully")
	fmt.Println()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupTestBucket()

	os.Exit(code)
}

func cleanupTestBucket() {
	if testBucket == "" || gcsClient == nil {
		return
	}

	ctx := context.Background()
	obj, err := gcs.Object(gcsClient, testBucket)
	if err != nil {
		return
	}

	// List and delete all objects in test path
	bucket, path, err := extractBucketAndPath(testBucket)
	if err != nil {
		return
	}

	it := gcsClient.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: path})
	for {
		attrs, err := it.Next()
		if err != nil {
			break
		}
		gcsClient.Bucket(bucket).Object(attrs.Name).Delete(ctx)
	}

	// Try to delete the test path index file
	obj.Delete(ctx)
}

func extractBucketAndPath(gcsURL string) (bucket, path string, err error) {
	obj, err := gcs.Object(gcsClient, gcsURL)
	if err != nil {
		return "", "", err
	}
	return obj.BucketName(), obj.ObjectName(), nil
}

func TestIntegration_CreateRepository(t *testing.T) {
	// Create a unique test path
	testPath := fmt.Sprintf("%s/create-repo-%d", testBucket, time.Now().Unix())

	// Create repository
	r, err := New(testPath, gcsClient)
	if err != nil {
		t.Fatalf("Failed to create repo object: %v", err)
	}

	err = Create(r)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Verify index.yaml exists
	indexURL := fmt.Sprintf("%s/index.yaml", testPath)
	obj, err := gcs.Object(gcsClient, indexURL)
	if err != nil {
		t.Fatalf("Failed to get index object: %v", err)
	}

	_, err = obj.Attrs(context.Background())
	if err != nil {
		t.Errorf("Index file not found: %v", err)
	}

	// Test idempotency - creating again should not error
	err = Create(r)
	if err != nil {
		t.Errorf("Create should be idempotent, but got error: %v", err)
	}
}

func TestIntegration_PushChart(t *testing.T) {
	// Create a unique test path
	testPath := fmt.Sprintf("%s/push-chart-%d", testBucket, time.Now().Unix())

	// Create repository
	r, err := New(testPath, gcsClient)
	if err != nil {
		t.Fatalf("Failed to create repo object: %v", err)
	}

	// Initialize repository entry (required for push)
	r.entry = &repo.Entry{
		Name: "test-repo",
		URL:  testPath,
	}

	err = Create(r)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Package test chart
	chartPath := "../../testdata/charts/test-chart"
	tmpDir := t.TempDir()

	// Use helm to package the chart
	chartInterface, err := loader.Load(chartPath)
	if err != nil {
		t.Fatalf("Failed to load test chart: %v", err)
	}

	chart, ok := chartInterface.(*chartv2.Chart)
	if !ok {
		t.Fatalf("Failed to convert chart to v2.Chart")
	}

	packagedPath := filepath.Join(tmpDir, fmt.Sprintf("%s-%s.tgz", chart.Metadata.Name, chart.Metadata.Version))
	err = packageChart(chartPath, tmpDir)
	if err != nil {
		t.Fatalf("Failed to package chart: %v", err)
	}

	// Push chart
	err = r.PushChart(packagedPath, false, false, false, "", "", nil)
	if err != nil {
		t.Fatalf("Failed to push chart: %v", err)
	}

	// Verify chart exists in GCS
	chartURL := fmt.Sprintf("%s/%s-%s.tgz", testPath, chart.Metadata.Name, chart.Metadata.Version)
	obj, err := gcs.Object(gcsClient, chartURL)
	if err != nil {
		t.Fatalf("Failed to get chart object: %v", err)
	}

	_, err = obj.Attrs(context.Background())
	if err != nil {
		t.Errorf("Chart file not found: %v", err)
	}

	// Verify index was updated
	indexURL := fmt.Sprintf("%s/index.yaml", testPath)
	indexObj, err := gcs.Object(gcsClient, indexURL)
	if err != nil {
		t.Fatalf("Failed to get index object: %v", err)
	}

	reader, err := indexObj.NewReader(context.Background())
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}
	defer reader.Close()

	// Verify chart is in index
	// (In a more complete test, we would parse the index and verify the entry)
}

func TestIntegration_RemoveChart(t *testing.T) {
	// Create a unique test path
	testPath := fmt.Sprintf("%s/remove-chart-%d", testBucket, time.Now().Unix())

	// Create and setup repository
	r, err := New(testPath, gcsClient)
	if err != nil {
		t.Fatalf("Failed to create repo object: %v", err)
	}

	r.entry = &repo.Entry{
		Name: "test-repo",
		URL:  testPath,
	}

	err = Create(r)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Package and push test chart
	chartPath := "../../testdata/charts/test-chart"
	tmpDir := t.TempDir()

	chartInterface, err := loader.Load(chartPath)
	if err != nil {
		t.Fatalf("Failed to load test chart: %v", err)
	}

	chart, ok := chartInterface.(*chartv2.Chart)
	if !ok {
		t.Fatalf("Failed to convert chart to v2.Chart")
	}

	packagedPath := filepath.Join(tmpDir, fmt.Sprintf("%s-%s.tgz", chart.Metadata.Name, chart.Metadata.Version))
	err = packageChart(chartPath, tmpDir)
	if err != nil {
		t.Fatalf("Failed to package chart: %v", err)
	}

	err = r.PushChart(packagedPath, false, false, false, "", "", nil)
	if err != nil {
		t.Fatalf("Failed to push chart: %v", err)
	}

	// Remove chart
	err = r.RemoveChart(chart.Metadata.Name, chart.Metadata.Version, false)
	if err != nil {
		t.Fatalf("Failed to remove chart: %v", err)
	}

	// Verify chart is deleted from GCS
	chartURL := fmt.Sprintf("%s/%s-%s.tgz", testPath, chart.Metadata.Name, chart.Metadata.Version)
	obj, err := gcs.Object(gcsClient, chartURL)
	if err != nil {
		t.Fatalf("Failed to get chart object: %v", err)
	}

	_, err = obj.Attrs(context.Background())
	if err == nil {
		t.Error("Chart file should have been deleted, but still exists")
	}
}

func TestIntegration_ConcurrentPush(t *testing.T) {
	// Test optimistic locking / concurrent update handling
	testPath := fmt.Sprintf("%s/concurrent-%d", testBucket, time.Now().Unix())

	// Create repository
	r, err := New(testPath, gcsClient)
	if err != nil {
		t.Fatalf("Failed to create repo object: %v", err)
	}

	r.entry = &repo.Entry{
		Name: "test-repo",
		URL:  testPath,
	}

	err = Create(r)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// This test would require creating two separate repo instances
	// and attempting to push simultaneously to trigger ErrIndexOutOfDate
	// For now, we just verify the retry mechanism works

	chartPath := "../../testdata/charts/test-chart"
	tmpDir := t.TempDir()

	chartInterface, err := loader.Load(chartPath)
	if err != nil {
		t.Fatalf("Failed to load test chart: %v", err)
	}

	chart, ok := chartInterface.(*chartv2.Chart)
	if !ok {
		t.Fatalf("Failed to convert chart to v2.Chart")
	}

	packagedPath := filepath.Join(tmpDir, fmt.Sprintf("%s-%s.tgz", chart.Metadata.Name, chart.Metadata.Version))
	err = packageChart(chartPath, tmpDir)
	if err != nil {
		t.Fatalf("Failed to package chart: %v", err)
	}

	// Push with retry enabled
	err = r.PushChart(packagedPath, false, true, false, "", "", nil)
	if err != nil {
		t.Fatalf("Failed to push chart with retry: %v", err)
	}
}

// Helper function to package a chart
func packageChart(chartPath, destDir string) error {
	// Use helm package command
	cmd := exec.Command("helm", "package", chartPath, "-d", destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to package chart: %w, output: %s", err, string(output))
	}
	return nil
}
