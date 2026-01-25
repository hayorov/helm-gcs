# Release Process

This document describes the release process for helm-gcs, including GPG key setup for signing plugin artifacts to support Helm 4 verification.

## Helm 4 Compatibility

Starting with Helm 4, plugin verification is enabled by default. When users install a plugin, Helm checks for `.prov` (provenance) files alongside the plugin archives. These are GPG signatures that verify the authenticity and integrity of the plugin.

**Without .prov files:**
```bash
$ helm plugin install https://github.com/hayorov/helm-gcs
Error: plugin source does not support verification. Use --verify=false to skip verification
```

**With .prov files (properly signed):**
```bash
$ helm plugin install https://github.com/hayorov/helm-gcs
Installing helm-gcs 0.6.2 ...
Installed plugin: gcs
```

## GPG Key Setup

### One-Time Setup (for Repository Maintainers)

1. **Generate a GPG key pair** (if you don't have one):

```bash
gpg --full-generate-key
```

Follow the prompts:
- Key type: RSA and RSA (default)
- Key size: 4096 bits
- Expiration: 0 (does not expire) or set an expiration date
- Real name: Your name or "Helm GCS Plugin"
- Email: Your email or project email

2. **Get your GPG key fingerprint**:

```bash
gpg --list-secret-keys --keyid-format LONG
```

Output example:
```
sec   rsa4096/1234567890ABCDEF 2026-01-18 [SC]
      ABCDEF1234567890ABCDEF1234567890ABCDEF12
uid                 [ultimate] Helm GCS Plugin <maintainer@example.com>
```

The fingerprint is the long hex string: `ABCDEF1234567890ABCDEF1234567890ABCDEF12`

3. **Export your GPG private key**:

```bash
# Export the private key (keep this secure!)
gpg --armor --export-secret-keys ABCDEF1234567890ABCDEF1234567890ABCDEF12
```

4. **Add GitHub Secrets** (Repository Settings > Secrets and variables > Actions > Secrets):

   - `GPG_PRIVATE_KEY`: Paste the entire output from step 3 (including `-----BEGIN PGP PRIVATE KEY BLOCK-----` and `-----END PGP PRIVATE KEY BLOCK-----`)
   - `GPG_PASSPHRASE`: Your GPG key passphrase (if you set one, otherwise leave empty)

5. **Publish your public key** (optional but recommended):

```bash
# Export public key
gpg --armor --export ABCDEF1234567890ABCDEF1234567890ABCDEF12 > helm-gcs-public-key.asc

# Upload to a key server (optional)
gpg --send-keys ABCDEF1234567890ABCDEF1234567890ABCDEF12
```

## Release Workflow

### Automated Release (Recommended)

Releases are automatically created when you push a new tag:

```bash
# Ensure your changes are committed
git add .
git commit -m "Release v0.x.x"

# Create and push a tag
git tag -a v0.x.x -m "Release v0.x.x"
git push origin v0.x.x
```

The GitHub Actions workflow will:
1. Run security scans (Trivy)
2. Import the GPG key from secrets
3. Build binaries for all platforms
4. Sign all archives with GPG (creates `.prov` files)
5. Create a GitHub release with all artifacts

### Manual Release (if needed)

If you need to create a release manually:

```bash
# Set your GPG fingerprint
export GPG_FINGERPRINT="ABCDEF1234567890ABCDEF1234567890ABCDEF12"

# Create release
goreleaser release --clean
```

This requires:
- GoReleaser installed: `brew install goreleaser` (macOS) or see [GoReleaser docs](https://goreleaser.com/install/)
- GitHub token with repo permissions
- GPG key configured locally

## Verifying Releases

### Verify a Release Locally

```bash
# Download a release archive and its .prov file
curl -LO https://github.com/hayorov/helm-gcs/releases/download/v0.6.2/helm-gcs_Darwin_arm64.tar.gz
curl -LO https://github.com/hayorov/helm-gcs/releases/download/v0.6.2/helm-gcs_Darwin_arm64.tar.gz.prov

# Import the public key (if not already imported)
curl -L https://github.com/hayorov/helm-gcs/releases/download/v0.6.2/helm-gcs-public-key.asc | gpg --import

# Verify the signature
gpg --verify helm-gcs_Darwin_arm64.tar.gz.prov helm-gcs_Darwin_arm64.tar.gz
```

Expected output:
```
gpg: Signature made ...
gpg: Good signature from "Helm GCS Plugin <...>"
```

### Test Helm 4 Installation

After a release, verify that Helm 4 can install the plugin without `--verify=false`:

```bash
# Remove existing plugin
helm plugin uninstall gcs

# Install with verification (should work without --verify=false)
helm plugin install https://github.com/hayorov/helm-gcs

# Verify installation
helm gcs version
```

## Troubleshooting

### Release Fails with GPG Error

**Error:** `gpg: signing failed: No secret key`

**Solution:** Ensure the `GPG_PRIVATE_KEY` secret is properly set in GitHub Actions secrets and includes the full key block.

### Users Can't Verify Plugin

**Error:** `gpg: Can't check signature: No public key`

**Solution:** Users need to import the public key first. Publish your public key:
1. In the GitHub release notes
2. On a key server (e.g., `keys.openpgp.org`)
3. In the repository (e.g., `keys/helm-gcs-public-key.asc`)

### Helm 4 Still Requires --verify=false

**Possible causes:**
1. `.prov` files weren't created in the release
   - Check: Look for `.prov` files in the GitHub release assets
   - Fix: Ensure `GPG_FINGERPRINT` environment variable is set in the workflow

2. `.prov` files are invalid
   - Check: Download and verify manually using `gpg --verify`
   - Fix: Verify GPG key is valid and not expired

3. User's Helm installation has issues
   - Check: `helm version` (should be 4.0.0+)
   - Fix: Update Helm to latest version

## Testing Changes

Before releasing, test the signing process locally:

```bash
# Create a snapshot release (doesn't publish)
export GPG_FINGERPRINT="ABCDEF1234567890ABCDEF1234567890ABCDEF12"
goreleaser release --snapshot --clean --skip=publish

# Verify .prov files were created
ls -la dist/*.prov

# Test signature verification
gpg --verify dist/helm-gcs_Darwin_arm64.tar.gz.prov dist/helm-gcs_Darwin_arm64.tar.gz
```

## References

- [Helm Plugin Documentation](https://helm.sh/docs/plugins/user/)
- [Helm Provenance and Integrity](https://helm.sh/docs/topics/provenance/)
- [GoReleaser Signing Documentation](https://goreleaser.com/customization/sign/)
- [GPG Documentation](https://www.gnupg.org/documentation/)
