$ErrorActionPreference = "Stop"

# Ensure we're in the plugin directory
if (-not $env:HELM_PLUGIN_DIR) {
    Write-Error "HELM_PLUGIN_DIR is not set"
    exit 1
}

Set-Location $env:HELM_PLUGIN_DIR

# Extract version from plugin.yaml
if (-not (Test-Path "plugin.yaml")) {
    Write-Error "plugin.yaml not found in $env:HELM_PLUGIN_DIR"
    exit 1
}

$versionLine = Select-String -Path "plugin.yaml" -Pattern 'version:' | Select-Object -First 1
if (-not $versionLine) {
    Write-Error "Could not extract version from plugin.yaml"
    exit 1
}
$version = ($versionLine.Line -replace '.*version:\s*"?([^"]+)"?.*', '$1').Trim()
if (-not $version) {
    Write-Error "Could not extract version from plugin.yaml"
    exit 1
}

# Detect Helm version using $HELM_BIN (set by Helm itself) to avoid
# picking up a different helm version that happens to be on PATH.
$helmBin = if ($env:HELM_BIN) { $env:HELM_BIN } else { "helm" }
$helmMajorVersion = ""
try {
    $helmVersionOutput = & $helmBin version --short 2>$null
    if ($helmVersionOutput -match 'v(\d+)') {
        $helmMajorVersion = $Matches[1]
    }
} catch {
    # helm not found or version failed; fall through
}

# For Helm 4, recommend using the separate plugin packages
if ($helmMajorVersion -eq "4") {
    Write-Host ""
    Write-Host "=========================================="
    Write-Host "  Helm 4 Detected"
    Write-Host "=========================================="
    Write-Host ""
    Write-Host "For Helm 4, we recommend installing the separate plugin packages"
    Write-Host "for better compatibility with the new plugin system:"
    Write-Host ""
    Write-Host "  # CLI plugin (helm gcs init/push/rm)"
    Write-Host "  helm plugin install https://github.com/hayorov/helm-gcs/releases/download/v${version}/helm-gcs-plugin.tar.gz"
    Write-Host ""
    Write-Host "  # Getter plugin (gs:// protocol support)"
    Write-Host "  helm plugin install https://github.com/hayorov/helm-gcs/releases/download/v${version}/helm-gcs-getter-plugin.tar.gz"
    Write-Host ""
    Write-Host "Continuing with legacy installation..."
    Write-Host ""
}

Write-Host "Installing helm-gcs ${version} ..."

# Detect architecture
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64"  { "x86_64" }
    "ARM64"  { "arm64" }
    default  {
        Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
        exit 1
    }
}

$baseUrl = "https://github.com/hayorov/helm-gcs/releases/download/v${version}"

if (Test-Path "bin") {
    Remove-Item -Recurse -Force "bin"
}
New-Item -ItemType Directory -Path "bin" | Out-Null

foreach ($binary in @("helm-gcs", "helm-gcs-getter")) {
    $filename = "${binary}_Windows_${arch}.zip"
    $url = "${baseUrl}/${filename}"

    Write-Host "Downloading from: ${url}"

    try {
        Invoke-WebRequest -Uri $url -OutFile $filename -UseBasicParsing
    } catch {
        Write-Error "Failed to download ${url}: $_"
        exit 1
    }

    try {
        Expand-Archive -Path $filename -DestinationPath "bin" -Force
    } catch {
        Remove-Item -Force $filename -ErrorAction SilentlyContinue
        Write-Error "Failed to extract ${filename}: $_"
        exit 1
    }

    Remove-Item -Force $filename -ErrorAction SilentlyContinue
}

# Verify installation
if (-not (Test-Path "bin\helm-gcs.exe")) {
    Write-Error "helm-gcs.exe binary not found after extraction"
    exit 1
}

if (-not (Test-Path "bin\helm-gcs-getter.exe")) {
    Write-Error "helm-gcs-getter.exe binary not found after extraction"
    exit 1
}

Write-Host ""
Write-Host "helm-gcs ${version} is correctly installed."
Write-Host ""
Write-Host "Usage:"
Write-Host "  helm gcs init gs://bucket/path              # Initialize repository"
Write-Host "  helm repo add repo-name gs://bucket/path    # Add repository to Helm"
Write-Host "  helm gcs push chart.tgz repo-name           # Push a chart"
Write-Host "  helm repo update                            # Update Helm cache"
Write-Host "  helm fetch repo-name/chart                  # Fetch a chart"
Write-Host ""
