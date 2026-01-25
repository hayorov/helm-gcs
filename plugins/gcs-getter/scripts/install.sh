#!/bin/sh

set -e

if [ -z "$HELM_PLUGIN_DIR" ]; then
    echo "Error: HELM_PLUGIN_DIR is not set"
    exit 1
fi

cd "$HELM_PLUGIN_DIR" || exit 1

if [ ! -f plugin.yaml ]; then
    echo "Error: plugin.yaml not found in $HELM_PLUGIN_DIR"
    exit 1
fi

version="$(grep 'version:' plugin.yaml | cut -d'"' -f2)"
if [ -z "$version" ]; then
    echo "Error: Could not extract version from plugin.yaml"
    exit 1
fi

echo "Installing helm-gcs-getter plugin ${version}..."

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

url="https://github.com/hayorov/helm-gcs/releases/download/v${version}/helm-gcs-getter_${os}_${arch}.tar.gz"
filename="helm-gcs-getter_${os}_${arch}.tar.gz"

echo "Downloading from: ${url}"

rm -rf bin && mkdir -p bin

if command -v curl > /dev/null 2>&1; then
    curl -sSL -o "$filename" "$url" || {
        echo "Error: Failed to download $url"
        exit 1
    }
elif command -v wget > /dev/null 2>&1; then
    wget -q -O "$filename" "$url" || {
        echo "Error: Failed to download $url"
        exit 1
    }
else
    echo "Error: curl or wget is required"
    exit 1
fi

tar xzf "$filename" -C bin || {
    echo "Error: Failed to extract $filename"
    rm -f "$filename"
    exit 1
}

rm -f "$filename"

if [ ! -x "bin/helm-gcs-getter" ] && [ ! -f "bin/helm-gcs-getter.exe" ]; then
    echo "Error: helm-gcs-getter binary not found"
    exit 1
fi

echo ""
echo "helm-gcs-getter plugin ${version} installed successfully."
echo ""
echo "You can now use gs:// URLs with Helm:"
echo "  helm repo add myrepo gs://bucket/path"
echo "  helm pull gs://bucket/path/chart-1.0.0.tgz"
echo ""
