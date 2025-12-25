# GitHub Actions Workflows

This directory contains CI/CD workflows for the helm-gcs project.

## Workflows

### test.yml
Runs on every pull request:
- Code formatting check
- Static analysis (go vet, golangci-lint)
- Unit tests with race detector
- Code coverage reporting
- Binary build verification

**No configuration required** - runs automatically on PRs.

### release.yml
Runs on git tags:
- Security scanning with Trivy
- Multi-platform builds (Linux, macOS, Windows on amd64/arm64)
- GitHub release creation

**No configuration required** - runs automatically when tags are pushed.

### integration-test.yml
Runs integration tests against real GCS bucket:
- Manual trigger via workflow_dispatch
- Scheduled weekly (Sundays at 2 AM UTC)

**Requires configuration** - see setup instructions below.

## Setting Up Integration Tests

Integration tests require GCP credentials and a GCS bucket.

### Prerequisites

1. A Google Cloud Platform (GCP) project
2. A GCS bucket for testing
3. Admin access to your GitHub repository

### Step 1: Create a GCS Bucket

```bash
# Set your project ID
export PROJECT_ID=your-gcp-project-id
gcloud config set project $PROJECT_ID

# Create a bucket for integration tests
export BUCKET_NAME=helm-gcs-integration-tests
gsutil mb -p $PROJECT_ID -l us-central1 gs://$BUCKET_NAME

# Verify bucket creation
gsutil ls gs://$BUCKET_NAME
```

### Step 2: Create a Service Account

```bash
# Create service account
gcloud iam service-accounts create helm-gcs-github-actions \
    --display-name="Helm GCS GitHub Actions" \
    --description="Service account for helm-gcs integration tests in GitHub Actions"

# Get the service account email
export SA_EMAIL=helm-gcs-github-actions@${PROJECT_ID}.iam.gserviceaccount.com

# Grant Storage Admin role to the service account
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/storage.admin"

# Alternative: Grant bucket-specific permissions (more secure)
gsutil iam ch serviceAccount:${SA_EMAIL}:roles/storage.objectAdmin gs://$BUCKET_NAME

# Create and download JSON key
gcloud iam service-accounts keys create gcs-test-key.json \
    --iam-account=$SA_EMAIL

# Display the key (you'll need this for GitHub)
cat gcs-test-key.json
```

**Important:** Keep this JSON key secure! Delete it from your local machine after adding to GitHub.

### Step 3: Configure GitHub Secrets

1. **Go to your GitHub repository**
   - Navigate to: Settings → Secrets and variables → Actions

2. **Add Secret: GCS_TEST_CREDENTIALS**
   - Click "New repository secret"
   - Name: `GCS_TEST_CREDENTIALS`
   - Value: Paste the **entire content** of `gcs-test-key.json`
   - Click "Add secret"

3. **Add Variable: GCS_TEST_BUCKET**
   - Click on "Variables" tab
   - Click "New repository variable"
   - Name: `GCS_TEST_BUCKET`
   - Value: `gs://helm-gcs-integration-tests/github-actions-tests`
   - Click "Add variable"

### Step 4: Clean Up Local Files

```bash
# IMPORTANT: Delete the service account key from your local machine
rm gcs-test-key.json
```

### Step 5: Verify Setup

1. **Manual Test Run**
   - Go to: Actions → integration-tests → Run workflow
   - Use default bucket or specify a custom one
   - Click "Run workflow"
   - Monitor the workflow execution

2. **Check Test Output**
   - Verify authentication step succeeds
   - Check that tests run against the GCS bucket
   - Confirm cleanup happens after tests

## Alternative: Using Workload Identity Federation (Recommended for Production)

For enhanced security, use Workload Identity Federation instead of service account keys:

```bash
# Create Workload Identity Pool
gcloud iam workload-identity-pools create "github-actions" \
    --project="${PROJECT_ID}" \
    --location="global" \
    --display-name="GitHub Actions Pool"

# Create Workload Identity Provider
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
    --project="${PROJECT_ID}" \
    --location="global" \
    --workload-identity-pool="github-actions" \
    --display-name="GitHub Provider" \
    --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
    --issuer-uri="https://token.actions.githubusercontent.com"

# Grant permissions
gcloud iam service-accounts add-iam-policy-binding "${SA_EMAIL}" \
    --project="${PROJECT_ID}" \
    --role="roles/iam.workloadIdentityUser" \
    --member="principalSet://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-actions/attribute.repository/YOUR_GITHUB_ORG/helm-gcs"
```

Then update the workflow to use Workload Identity Federation instead of JSON keys.

## Troubleshooting

### Integration Tests Not Running

**Problem:** Workflow doesn't start when manually triggered or on schedule.

**Solution:**
- Check that `GCS_TEST_BUCKET` variable is set (not empty)
- Verify the workflow condition: `if: vars.GCS_TEST_BUCKET != ''`

### Authentication Failures

**Problem:** "Error: google-github-actions/auth failed with: retry function failed after 3 attempts"

**Solutions:**
1. Verify `GCS_TEST_CREDENTIALS` secret contains valid JSON
2. Check that the service account still exists in GCP
3. Ensure the service account key hasn't expired
4. Verify the service account has the correct permissions

### Permission Denied Errors

**Problem:** "403 Forbidden" or "Permission denied" when accessing GCS

**Solutions:**
1. Check service account has Storage Admin role:
   ```bash
   gcloud projects get-iam-policy $PROJECT_ID \
       --flatten="bindings[].members" \
       --filter="bindings.members:serviceAccount:${SA_EMAIL}"
   ```

2. Verify bucket-specific permissions:
   ```bash
   gsutil iam get gs://$BUCKET_NAME
   ```

3. Grant necessary permissions:
   ```bash
   gsutil iam ch serviceAccount:${SA_EMAIL}:roles/storage.objectAdmin gs://$BUCKET_NAME
   ```

### Tests Timeout

**Problem:** Integration tests exceed 10-minute timeout.

**Solutions:**
- Check GCS bucket location (use same region as GitHub Actions runners for faster access)
- Verify network connectivity to GCS
- Review test logs for hanging operations

## Security Best Practices

1. **Principle of Least Privilege**
   - Grant only necessary permissions (Storage Object Admin instead of Storage Admin when possible)
   - Use bucket-specific permissions rather than project-wide

2. **Rotate Keys Regularly**
   - Service account keys don't expire by default
   - Rotate them every 90 days
   - Consider using Workload Identity Federation (no keys needed)

3. **Audit Access**
   - Review service account usage in GCP Console
   - Monitor logs for unusual activity
   - Use Cloud Asset Inventory to track permissions

4. **Separate Environments**
   - Use different buckets for different branches/environments
   - Never use production buckets for testing

## Cost Management

Integration tests incur GCP costs:
- Storage: ~$0.02/GB/month for standard storage
- Operations: ~$0.005 per 10,000 operations
- Network: Usually minimal for same-region access

**Estimated cost:** < $1/month for typical usage

To minimize costs:
- Run integration tests on-demand instead of every commit
- Use lifecycle rules to auto-delete old test data:
  ```bash
  gsutil lifecycle set lifecycle.json gs://$BUCKET_NAME
  ```

Example `lifecycle.json`:
```json
{
  "lifecycle": {
    "rule": [{
      "action": {"type": "Delete"},
      "condition": {"age": 7}
    }]
  }
}
```

## References

- [GitHub Actions documentation](https://docs.github.com/en/actions)
- [google-github-actions/auth](https://github.com/google-github-actions/auth)
- [GCP Service Accounts](https://cloud.google.com/iam/docs/service-accounts)
- [GCS IAM Permissions](https://cloud.google.com/storage/docs/access-control/iam-permissions)
