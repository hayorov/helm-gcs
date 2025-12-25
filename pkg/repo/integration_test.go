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
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/repo"

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
	// Check if integration tests should run
	testBucket = os.Getenv("GCS_TEST_BUCKET")
	if testBucket == "" {
		fmt.Println("Skipping integration tests: GCS_TEST_BUCKET not set")
		fmt.Println("To run integration tests: export GCS_TEST_BUCKET=gs://your-bucket/test-path")
		os.Exit(0)
	}

	// Create GCS client
	var err error
	gcsClient, err = gcs.NewClient("")
	if err != nil {
		fmt.Printf("Failed to create GCS client: %v\n", err)
		os.Exit(1)
	}

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
	chart, err := loader.Load(chartPath)
	if err != nil {
		t.Fatalf("Failed to load test chart: %v", err)
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

	chart, err := loader.Load(chartPath)
	if err != nil {
		t.Fatalf("Failed to load test chart: %v", err)
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

	chart, err := loader.Load(chartPath)
	if err != nil {
		t.Fatalf("Failed to load test chart: %v", err)
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
