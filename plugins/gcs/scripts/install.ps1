$ErrorActionPreference = "Stop"

if (-not $env:HELM_PLUGIN_DIR) {
    Write-Error "HELM_PLUGIN_DIR is not set"
    exit 1
}

Set-Location $env:HELM_PLUGIN_DIR

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

Write-Host "Installing helm-gcs CLI plugin ${version}..."

# Detect architecture
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64"  { "x86_64" }
    "ARM64"  { "arm64" }
    default  {
        Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
        exit 1
    }
}

$url = "https://github.com/hayorov/helm-gcs/releases/download/v${version}/helm-gcs_Windows_${arch}.zip"
$filename = "helm-gcs_Windows_${arch}.zip"

Write-Host "Downloading from: ${url}"

if (Test-Path "bin") {
    Remove-Item -Recurse -Force "bin"
}
New-Item -ItemType Directory -Path "bin" | Out-Null

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

if (-not (Test-Path "bin\helm-gcs.exe")) {
    Write-Error "helm-gcs.exe binary not found after extraction"
    exit 1
}

Write-Host ""
Write-Host "helm-gcs CLI plugin ${version} installed successfully."
Write-Host ""
Write-Host "Usage:"
Write-Host "  helm gcs init gs://bucket/path       # Initialize repository"
Write-Host "  helm gcs push chart.tgz repo-name    # Push a chart"
Write-Host "  helm gcs rm chart repo-name          # Remove a chart"
Write-Host ""
