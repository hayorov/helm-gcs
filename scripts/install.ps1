$ErrorActionPreference = "Stop"

# Ensure we're in the plugin directory
if (-not $env:HELM_PLUGIN_DIR) {
    Write-Error "HELM_PLUGIN_DIR is not set"
    exit 1
}

Set-Location $env:HELM_PLUGIN_DIR

# Extract version from plugin.yaml
# On Helm 4 the install hook runs after plugin.yaml is renamed, so check .bak too
$pluginFile = if (Test-Path "plugin.yaml") { "plugin.yaml" }
              elseif (Test-Path "plugin.yaml.bak") { "plugin.yaml.bak" }
              else { $null }
if (-not $pluginFile) {
    Write-Error "plugin.yaml not found in $env:HELM_PLUGIN_DIR"
    exit 1
}

$versionLine = Select-String -Path $pluginFile -Pattern 'version:' | Select-Object -First 1
if (-not $versionLine) {
    Write-Error "Could not extract version from $pluginFile"
    exit 1
}
$version = ($versionLine.Line -replace '.*version:\s*"?([^"]+)"?.*', '$1').Trim()
if (-not $version) {
    Write-Error "Could not extract version from $pluginFile"
    exit 1
}

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

function Download-Binary {
    param(
        [string]$Binary,
        [string]$Dest
    )

    if (-not (Test-Path $Dest)) {
        New-Item -ItemType Directory -Path $Dest -Force | Out-Null
    }

    $filename = "${Binary}_Windows_${arch}.zip"
    $url = "${baseUrl}/${filename}"
    $outPath = Join-Path $Dest $filename

    Write-Host "Downloading from: ${url}"

    try {
        Invoke-WebRequest -Uri $url -OutFile $outPath -UseBasicParsing
    } catch {
        Write-Error "Failed to download ${url}: $_"
        exit 1
    }

    try {
        Expand-Archive -Path $outPath -DestinationPath $Dest -Force
    } catch {
        Remove-Item -Force $outPath -ErrorAction SilentlyContinue
        Write-Error "Failed to extract ${filename}: $_"
        exit 1
    }

    Remove-Item -Force $outPath -ErrorAction SilentlyContinue

    $exePath = Join-Path $Dest "${Binary}.exe"
    if (-not (Test-Path $exePath)) {
        Write-Error "${Binary}.exe not found after extraction in ${Dest}"
        exit 1
    }
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

if ($helmMajorVersion -eq "4") {
    Write-Host "Helm 4 detected -- installing sub-plugins for full cli + getter support..."

    $pluginsDir = Split-Path $env:HELM_PLUGIN_DIR -Parent

    $entries = @(
        @{ SrcDir = "gcs";        Binary = "helm-gcs";        DestName = "helm-gcs-plugin" },
        @{ SrcDir = "gcs-getter"; Binary = "helm-gcs-getter"; DestName = "helm-gcs-getter-plugin" }
    )

    foreach ($entry in $entries) {
        $src  = Join-Path $env:HELM_PLUGIN_DIR "plugins\$($entry.SrcDir)"
        $dest = Join-Path $pluginsDir $entry.DestName

        if (-not (Test-Path $src)) {
            Write-Error "Sub-plugin source not found: $src"
            exit 1
        }

        if (Test-Path $dest) {
            Remove-Item -Recurse -Force $dest
        }
        Copy-Item -Recurse $src $dest

        $binDir = Join-Path $dest "bin"
        Download-Binary -Binary $entry.Binary -Dest $binDir
    }

    # Disable the root plugin so Helm 4 doesn't see a duplicate "gcs" plugin
    $rootYaml = Join-Path $env:HELM_PLUGIN_DIR "plugin.yaml"
    if (Test-Path $rootYaml) {
        Rename-Item $rootYaml "plugin.yaml.bak" -Force -ErrorAction SilentlyContinue
    }

    Write-Host ""
    Write-Host "helm-gcs ${version} installed for Helm 4."
    Write-Host ""
    Write-Host "  helm gcs init gs://bucket/path         # Initialize repository"
    Write-Host "  helm repo add myrepo gs://bucket/path   # Add repository"
    Write-Host "  helm gcs push chart.tgz myrepo          # Push a chart"
    Write-Host "  helm gcs rm chart myrepo                # Remove a chart"
    Write-Host ""
    exit 0
}

# Helm 3: legacy single-plugin install (both binaries into this plugin's bin/)
Write-Host "Installing helm-gcs ${version} ..."

if (Test-Path "bin") {
    Remove-Item -Recurse -Force "bin"
}
New-Item -ItemType Directory -Path "bin" | Out-Null

foreach ($binary in @("helm-gcs", "helm-gcs-getter")) {
    Download-Binary -Binary $binary -Dest "bin"
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
