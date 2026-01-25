#!/bin/sh
# Install both helm-gcs plugins for Helm 4
# Usage: curl -fsSL https://raw.githubusercontent.com/hayorov/helm-gcs/master/scripts/install-helm4.sh | sh

set -e

VERSION="${HELM_GCS_VERSION:-0.7.0}"
REPO="https://github.com/hayorov/helm-gcs"

# Detect OS
os=$(uname -s)
case "$os" in
    Linux*)   os="Linux" ;;
    Darwin*)  os="Darwin" ;;
    MINGW*|MSYS*|CYGWIN*) os="Windows" ;;
    *)
        echo "Unsupported OS: $os"
        exit 1
        ;;
esac

# Detect architecture
arch=$(uname -m)
case "$arch" in
    x86_64)         arch="x86_64" ;;
    aarch64|arm64)  arch="arm64" ;;
    *)
        echo "Unsupported architecture: $arch"
        exit 1
        ;;
esac

echo "Installing helm-gcs plugins v${VERSION} for ${os}/${arch}..."

# Get Helm plugin directory
HELM_PLUGINS="${HELM_PLUGINS:-$(helm env HELM_PLUGINS 2>/dev/null || echo "$HOME/.local/share/helm/plugins")}"

install_plugin() {
    name="$1"
    binary="$2"
    type="$3"
    
    plugin_dir="${HELM_PLUGINS}/${name}"
    
    echo ""
    echo "Installing ${name} plugin..."
    
    # Remove existing
    rm -rf "$plugin_dir"
    mkdir -p "$plugin_dir/bin"
    
    # Download binary
    url="${REPO}/releases/download/v${VERSION}/${binary}_${os}_${arch}.tar.gz"
    echo "  Downloading from: ${url}"
    
    if command -v curl > /dev/null 2>&1; then
        curl -fsSL "$url" | tar xz -C "$plugin_dir/bin"
    elif command -v wget > /dev/null 2>&1; then
        wget -qO- "$url" | tar xz -C "$plugin_dir/bin"
    else
        echo "Error: curl or wget is required"
        exit 1
    fi
    
    # Create plugin.yaml
    if [ "$type" = "cli" ]; then
        cat > "$plugin_dir/plugin.yaml" << EOF
apiVersion: v1
type: cli/v1
name: ${name}
version: "${VERSION}"
runtime: subprocess
sourceURL: ${REPO}

config:
  usage: "gcs <command> [flags]"
  shortHelp: "Manage Helm chart repositories on Google Cloud Storage"

runtimeConfig:
  platformCommand:
    - command: "\${HELM_PLUGIN_DIR}/bin/${binary}"
EOF
    else
        cat > "$plugin_dir/plugin.yaml" << EOF
apiVersion: v1
type: getter/v1
name: ${name}
version: "${VERSION}"
runtime: subprocess
sourceURL: ${REPO}

config:
  protocols:
    - gs

runtimeConfig:
  platformCommand:
    - command: "\${HELM_PLUGIN_DIR}/bin/${binary}"
EOF
    fi
    
    echo "  Installed to: ${plugin_dir}"
}

# Install both plugins
install_plugin "gcs" "helm-gcs" "cli"
install_plugin "gcs-getter" "helm-gcs-getter" "getter"

echo ""
echo "=========================================="
echo "  helm-gcs v${VERSION} installed!"
echo "=========================================="
echo ""
echo "Installed plugins:"
helm plugin list 2>/dev/null | grep -E "^(NAME|gcs)" || true
echo ""
echo "Usage:"
echo "  helm gcs init gs://bucket/charts    # Initialize repository"
echo "  helm repo add myrepo gs://bucket/charts"
echo "  helm gcs push chart.tgz myrepo      # Push a chart"
echo ""
