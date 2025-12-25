# GitHub Actions Integration Tests Setup Guide

This guide walks you through setting up GCP integration tests in GitHub Actions using **best practices**.

## Table of Contents

- [Overview](#overview)
- [Method 1: Service Account Key (Quick Setup)](#method-1-service-account-key-quick-setup)
- [Method 2: Workload Identity Federation (Recommended)](#method-2-workload-identity-federation-recommended)
- [GitHub Configuration](#github-configuration)
- [Testing the Setup](#testing-the-setup)
- [Troubleshooting](#troubleshooting)

---

## Overview

### What You Need

- **GCP Project**: Your Google Cloud project (e.g., `hayorov`)
- **GCS Bucket**: A bucket for integration tests (e.g., `gs://gcs-helm`)
- **Service Account**: With permissions to access the bucket
- **GitHub Repository**: Admin access to configure secrets

### Authentication Methods Comparison

| Feature | Service Account Key | Workload Identity Federation |
|---------|-------------------|------------------------------|
| **Security** | 🟡 Medium (rotating keys needed) | 🟢 High (no keys to manage) |
| **Setup Complexity** | 🟢 Easy (5 minutes) | 🟡 Medium (15 minutes) |
| **Maintenance** | 🟡 Manual rotation needed | 🟢 Zero maintenance |
| **Best Practice** | ⚠️ OK for small projects | ✅ **Recommended** |
| **Key Expiration** | ❌ Never (security risk) | ✅ No keys used |

**Recommendation**: Use Workload Identity Federation for production. Service Account Key is fine for getting started.

---

## Method 1: Service Account Key (Quick Setup)

### Step 1: Create Service Account

```bash
# Set your GCP project
export PROJECT_ID=hayorov
gcloud config set project $PROJECT_ID

# Create service account for GitHub Actions
gcloud iam service-accounts create helm-gcs-github-ci \
    --display-name="Helm GCS GitHub Actions CI" \
    --description="Service account for helm-gcs integration tests in GitHub Actions"

# Get the service account email
export SA_EMAIL=helm-gcs-github-ci@${PROJECT_ID}.iam.gserviceaccount.com
echo "Service Account: $SA_EMAIL"
```

### Step 2: Grant Permissions

**Option A: Project-Wide Storage Admin** (easier but broader permissions)

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/storage.admin"
```

**Option B: Bucket-Specific Permissions** (more secure, recommended)

```bash
# Grant Storage Object Admin on specific bucket
gsutil iam ch serviceAccount:${SA_EMAIL}:roles/storage.objectAdmin gs://gcs-helm

# Verify permissions
gsutil iam get gs://gcs-helm | grep $SA_EMAIL
```

### Step 3: Create JSON Key

```bash
# Create key file
gcloud iam service-accounts keys create ~/gcs-github-ci-key.json \
    --iam-account=$SA_EMAIL

# Display the key (you'll copy this to GitHub)
cat ~/gcs-github-ci-key.json

# IMPORTANT: This is sensitive! Copy it now and then delete the file
```

### Step 4: Configure GitHub Secrets

1. **Go to your GitHub repository**
   - Navigate to: `Settings` → `Secrets and variables` → `Actions`

2. **Add Secret: GCS_TEST_CREDENTIALS**
   - Click `New repository secret`
   - Name: `GCS_TEST_CREDENTIALS`
   - Value: Paste the **entire content** of `gcs-github-ci-key.json`
   - Click `Add secret`

3. **Add Variable: GCS_TEST_BUCKET**
   - Click on `Variables` tab
   - Click `New repository variable`
   - Name: `GCS_TEST_BUCKET`
   - Value: `gs://gcs-helm/helm-gcs-ci-tests`
   - Click `Add variable`

### Step 5: Clean Up Local Files

```bash
# CRITICAL: Delete the key file from your machine
rm ~/gcs-github-ci-key.json

# Verify it's deleted
ls ~/gcs-github-ci-key.json  # Should say "No such file"
```

### Step 6: Security Best Practices

```bash
# List all keys for this service account (for auditing)
gcloud iam service-accounts keys list \
    --iam-account=$SA_EMAIL

# Set up key rotation reminder (recommended: rotate every 90 days)
# Add to your calendar: "Rotate GitHub CI GCP keys"
```

---

## Method 2: Workload Identity Federation (Recommended)

This method is **more secure** as it doesn't use long-lived keys. GitHub Actions authenticates directly to GCP.

### Step 1: Enable Required APIs

```bash
export PROJECT_ID=hayorov
gcloud config set project $PROJECT_ID

# Enable required APIs
gcloud services enable iamcredentials.googleapis.com
gcloud services enable cloudresourcemanager.googleapis.com
gcloud services enable sts.googleapis.com
```

### Step 2: Create Service Account

```bash
# Create service account
gcloud iam service-accounts create helm-gcs-github-wif \
    --display-name="Helm GCS GitHub WIF" \
    --description="Workload Identity Federation for helm-gcs GitHub Actions"

export SA_EMAIL=helm-gcs-github-wif@${PROJECT_ID}.iam.gserviceaccount.com

# Grant bucket permissions
gsutil iam ch serviceAccount:${SA_EMAIL}:roles/storage.objectAdmin gs://gcs-helm
```

### Step 3: Create Workload Identity Pool

```bash
# Get your project number
export PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
echo "Project Number: $PROJECT_NUMBER"

# Create Workload Identity Pool
gcloud iam workload-identity-pools create "github-actions" \
    --project="${PROJECT_ID}" \
    --location="global" \
    --display-name="GitHub Actions Pool"

# Create Workload Identity Provider for GitHub
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
    --project="${PROJECT_ID}" \
    --location="global" \
    --workload-identity-pool="github-actions" \
    --display-name="GitHub Provider" \
    --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner" \
    --issuer-uri="https://token.actions.githubusercontent.com"
```

### Step 4: Configure IAM Bindings

```bash
# Replace YOUR_GITHUB_ORG with your GitHub username or organization
export GITHUB_REPO="hayorov/helm-gcs"

# Allow GitHub Actions from your repository to impersonate the service account
gcloud iam service-accounts add-iam-policy-binding "${SA_EMAIL}" \
    --project="${PROJECT_ID}" \
    --role="roles/iam.workloadIdentityUser" \
    --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github-actions/attribute.repository/${GITHUB_REPO}"

# Verify the binding
gcloud iam service-accounts get-iam-policy $SA_EMAIL
```

### Step 5: Get Workload Identity Provider Name

```bash
# Get the full provider name (you'll need this for GitHub)
export WIF_PROVIDER="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github-actions/providers/github-provider"
echo "Workload Identity Provider:"
echo $WIF_PROVIDER

# Save this - you'll add it to GitHub
```

### Step 6: Configure GitHub Secrets

1. **Go to GitHub repository settings**
   - Navigate to: `Settings` → `Secrets and variables` → `Actions`

2. **Add Secret: WIF_PROVIDER**
   - Click `New repository secret`
   - Name: `WIF_PROVIDER`
   - Value: The output from `echo $WIF_PROVIDER` above
   - Example: `projects/123456789/locations/global/workloadIdentityPools/github-actions/providers/github-provider`
   - Click `Add secret`

3. **Add Secret: WIF_SERVICE_ACCOUNT**
   - Click `New repository secret`
   - Name: `WIF_SERVICE_ACCOUNT`
   - Value: `helm-gcs-github-wif@hayorov.iam.gserviceaccount.com`
   - Click `Add secret`

4. **Add Variable: GCS_TEST_BUCKET**
   - Click on `Variables` tab
   - Click `New repository variable`
   - Name: `GCS_TEST_BUCKET`
   - Value: `gs://gcs-helm/helm-gcs-ci-tests`
   - Click `Add variable`

---

## GitHub Configuration

### Required Secrets

| Secret Name | Description | Example | When Used |
|------------|-------------|---------|-----------|
| `GCS_TEST_CREDENTIALS` | Service account JSON key | `{"type": "service_account"...}` | Method 1 only |
| `WIF_PROVIDER` | Workload Identity Provider resource name | `projects/123.../providers/github-provider` | Method 2 only |
| `WIF_SERVICE_ACCOUNT` | Service account email for WIF | `sa-name@project.iam.gserviceaccount.com` | Method 2 only |

### Required Variables

| Variable Name | Description | Example |
|--------------|-------------|---------|
| `GCS_TEST_BUCKET` | GCS bucket path for tests | `gs://gcs-helm/helm-gcs-ci-tests` |

### Optional Variables

| Variable Name | Description | Default |
|--------------|-------------|---------|
| `GCP_PROJECT_ID` | GCP project ID | Inferred from credentials |
| `GCS_TEST_REGION` | Bucket region | `us-central1` |

---

## Testing the Setup

### Test Service Account Locally

```bash
# Test with the service account (Method 1)
gcloud auth activate-service-account --key-file=~/gcs-github-ci-key.json
gsutil ls gs://gcs-helm/

# Test permissions
gsutil cp README.md gs://gcs-helm/test-write.txt
gsutil rm gs://gcs-helm/test-write.txt

echo "✓ Service account has correct permissions"
```

### Test GitHub Actions Workflow

1. **Manual Trigger**:
   - Go to: `Actions` → `integration-tests` → `Run workflow`
   - Click `Run workflow` button
   - Monitor the execution

2. **Check for Success**:
   - ✅ Authentication step succeeds
   - ✅ GCS access verification passes
   - ✅ Tests run and pass
   - ✅ Cleanup happens

3. **View Test Results**:
   - Click on the workflow run
   - Expand each step to see detailed logs

### Verify Permissions

```bash
# Check what the service account can do
gcloud projects get-iam-policy $PROJECT_ID \
    --flatten="bindings[].members" \
    --filter="bindings.members:serviceAccount:${SA_EMAIL}"

# Check bucket-specific permissions
gsutil iam get gs://gcs-helm | grep -A5 $SA_EMAIL
```

---

## Troubleshooting

### Common Issues

#### 1. "Permission Denied" Errors

**Problem**: `403 Forbidden` or `AccessDeniedException`

**Solutions**:

```bash
# Check service account permissions
gsutil iam get gs://gcs-helm

# Grant necessary permissions
gsutil iam ch serviceAccount:${SA_EMAIL}:roles/storage.objectAdmin gs://gcs-helm

# For project-wide access
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/storage.admin"
```

#### 2. "Invalid Credentials" in GitHub Actions

**Problem**: Authentication fails in CI

**Check**:
1. Verify secret is set correctly in GitHub
2. Check for extra whitespace in secret value
3. Ensure JSON is valid (use `jq` locally)

```bash
# Validate JSON locally
cat ~/gcs-github-ci-key.json | jq .
# Should parse without errors
```

#### 3. Workload Identity Federation Errors

**Problem**: `Error: google-github-actions/auth failed`

**Solutions**:

```bash
# Verify WIF pool exists
gcloud iam workload-identity-pools describe github-actions \
    --location=global

# Check IAM binding
gcloud iam service-accounts get-iam-policy $SA_EMAIL

# Verify the repository constraint
# Make sure GITHUB_REPO matches exactly: "owner/repo"
```

#### 4. Tests Run but Bucket Access Fails

**Problem**: Tests start but can't access GCS

**Check**:

```bash
# Verify bucket exists
gsutil ls gs://gcs-helm/

# Check bucket location
gsutil ls -L gs://gcs-helm/ | grep Location

# Ensure service account has access
gsutil iam ch serviceAccount:${SA_EMAIL}:roles/storage.objectAdmin gs://gcs-helm
```

---

## Security Recommendations

### 🔒 Best Practices

1. **Use Workload Identity Federation** for production
   - No long-lived keys to manage
   - Automatic key rotation
   - Better audit trail

2. **Principle of Least Privilege**
   ```bash
   # ✅ Good: Bucket-specific permissions
   gsutil iam ch serviceAccount:${SA_EMAIL}:objectAdmin gs://bucket

   # ⚠️ Avoid: Project-wide admin
   # Only use if absolutely necessary
   ```

3. **Separate Environments**
   ```bash
   # Use different buckets for different purposes
   gs://gcs-helm-ci-tests/        # GitHub Actions
   gs://gcs-helm-dev/             # Local development
   gs://gcs-helm-staging/         # Staging tests
   ```

4. **Regular Auditing**
   ```bash
   # List all service account keys (should be minimal)
   gcloud iam service-accounts keys list --iam-account=$SA_EMAIL

   # Review IAM bindings
   gcloud projects get-iam-policy $PROJECT_ID
   ```

5. **Key Rotation Schedule** (if using Method 1)
   - Rotate keys every **90 days**
   - Set calendar reminders
   - Document rotation procedure

6. **Monitor Usage**
   ```bash
   # Enable audit logs
   gcloud logging read "protoPayload.authenticationInfo.principalEmail=$SA_EMAIL" \
       --limit 50 --format json
   ```

### 🚫 What NOT to Do

- ❌ Never commit service account keys to git
- ❌ Don't use personal accounts for CI/CD
- ❌ Avoid overly broad permissions (e.g., Owner role)
- ❌ Don't share keys between environments
- ❌ Never log or echo credential values in CI

---

## Cost Management

### Estimated Costs

- **Storage**: ~$0.02/GB/month
- **Operations**: ~$0.005 per 10,000 operations
- **Network**: Minimal (same-region access)

**Estimated monthly cost**: < $1 for typical usage

### Cost Optimization

```bash
# Set lifecycle policy to auto-delete old test data
cat > lifecycle.json <<EOF
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "Delete"},
        "condition": {
          "age": 7,
          "matchesPrefix": ["helm-gcs-ci-tests/"]
        }
      }
    ]
  }
}
EOF

# Apply lifecycle policy
gsutil lifecycle set lifecycle.json gs://gcs-helm
```

---

## Additional Resources

- [Google Cloud IAM Documentation](https://cloud.google.com/iam/docs)
- [Workload Identity Federation Guide](https://cloud.google.com/iam/docs/workload-identity-federation)
- [google-github-actions/auth](https://github.com/google-github-actions/auth)
- [GCS IAM Permissions](https://cloud.google.com/storage/docs/access-control/iam-permissions)
- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)

---

## Quick Reference Commands

```bash
# View all service accounts
gcloud iam service-accounts list

# Check permissions on bucket
gsutil iam get gs://gcs-helm

# Test authentication locally
gcloud auth activate-service-account --key-file=key.json
gsutil ls gs://gcs-helm

# Delete service account (if needed)
gcloud iam service-accounts delete $SA_EMAIL

# Revoke access
gsutil iam ch -d serviceAccount:${SA_EMAIL}:roles/storage.objectAdmin gs://gcs-helm
```
