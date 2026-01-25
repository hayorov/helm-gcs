#!/bin/sh

set -e  # Exit on any error

# Ensure we're in the plugin directory
if [ -z "$HELM_PLUGIN_DIR" ]; then
    echo "Error: HELM_PLUGIN_DIR is not set"
    exit 1
fi

cd "$HELM_PLUGIN_DIR" || {
    echo "Error: Cannot change to plugin directory: $HELM_PLUGIN_DIR"
    exit 1
}

# Extract version from plugin.yaml
if [ ! -f plugin.yaml ]; then
    echo "Error: plugin.yaml not found in $HELM_PLUGIN_DIR"
    exit 1
fi

version="$(grep "version" plugin.yaml | cut -d '"' -f 2)"
if [ -z "$version" ]; then
    echo "Error: Could not extract version from plugin.yaml"
    exit 1
fi

echo "Installing helm-gcs ${version} ..."

# Detect OS
unameOut="$(uname -s)"
case "${unameOut}" in
    Linux*)             os=Linux;;
    Darwin*)            os=Darwin;;
    CYGWIN*)            os=Cygwin;;
    MINGW*|MSYS_NT*)    os=windows;;
    *)
        echo "Unsupported OS: ${unameOut}"
        exit 1
        ;;
esac

# Detect architecture
arch="$(uname -m)"
case "${arch}" in
    aarch64)    arch=arm64;;
    x86_64)     arch=x86_64;;
    arm64)      arch=arm64;;
    *)
        echo "Unsupported architecture: ${arch}"
        exit 1
        ;;
esac

url="https://github.com/hayorov/helm-gcs/releases/download/${version}/helm-gcs_${os}_${arch}.tar.gz"
filename="helm-gcs_${os}_${arch}.tar.gz"

echo "Downloading from: ${url}"

# Download archive
if command -v curl > /dev/null 2>&1; then
    if ! curl -sSL -o "$filename" "$url"; then
        echo "Error: Failed to download $url"
        exit 1
    fi
elif command -v wget > /dev/null 2>&1; then
    if ! wget -q -O "$filename" "$url"; then
        echo "Error: Failed to download $url"
        exit 1
    fi
else
    echo "Error: curl or wget is required"
    exit 1
fi

# Verify download
if [ ! -f "$filename" ]; then
    echo "Error: Downloaded file not found: $filename"
    exit 1
fi

# Install binary
rm -rf bin
mkdir -p bin

if ! tar xzf "$filename" -C bin; then
    echo "Error: Failed to extract $filename"
    rm -f "$filename"
    exit 1
fi

rm -f "$filename"

# Verify installation
if [ ! -x "bin/helm-gcs" ]; then
    echo "Error: helm-gcs binary not found or not executable"
    exit 1
fi

echo ""
echo "helm-gcs ${version} is correctly installed."
echo ""
echo "Usage:"
echo "  helm gcs init gs://bucket/path              # Initialize repository"
echo "  helm repo add repo-name gs://bucket/path    # Add repository to Helm"
echo "  helm gcs push chart.tgz repo-name           # Push a chart"
echo "  helm repo update                            # Update Helm cache"
echo "  helm fetch repo-name/chart                  # Fetch a chart"
echo ""
