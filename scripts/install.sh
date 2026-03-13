#!/bin/sh

set -e

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

version="$(grep "version" plugin.yaml | head -1 | cut -d '"' -f 2)"
if [ -z "$version" ]; then
    echo "Error: Could not extract version from plugin.yaml"
    exit 1
fi

# Detect OS
unameOut="$(uname -s)"
case "${unameOut}" in
    Linux*)             os=Linux;;
    Darwin*)            os=Darwin;;
    CYGWIN*)            os=Cygwin;;
    MINGW*|MSYS_NT*)    os=Windows;;
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

if [ "$os" = "Windows" ]; then
    ext="zip"
else
    ext="tar.gz"
fi

base_url="https://github.com/hayorov/helm-gcs/releases/download/v${version}"

# Download and extract a binary into a target directory
download_binary() {
    binary="$1"
    dest="$2"

    mkdir -p "$dest"
    filename="${binary}_${os}_${arch}.${ext}"
    url="${base_url}/${filename}"

    echo "Downloading from: ${url}"

    if command -v curl > /dev/null 2>&1; then
        if ! curl -sSL -o "${dest}/${filename}" "$url"; then
            echo "Error: Failed to download $url"
            exit 1
        fi
    elif command -v wget > /dev/null 2>&1; then
        if ! wget -q -O "${dest}/${filename}" "$url"; then
            echo "Error: Failed to download $url"
            exit 1
        fi
    else
        echo "Error: curl or wget is required"
        exit 1
    fi

    if [ ! -f "${dest}/${filename}" ]; then
        echo "Error: Downloaded file not found: ${dest}/${filename}"
        exit 1
    fi

    if [ "$ext" = "zip" ]; then
        if ! unzip -q -o "${dest}/${filename}" -d "$dest"; then
            echo "Error: Failed to extract ${filename}"
            rm -f "${dest}/${filename}"
            exit 1
        fi
    else
        if ! tar xzf "${dest}/${filename}" -C "$dest"; then
            echo "Error: Failed to extract ${filename}"
            rm -f "${dest}/${filename}"
            exit 1
        fi
    fi

    rm -f "${dest}/${filename}"

    if [ ! -x "${dest}/${binary}" ] && [ ! -f "${dest}/${binary}.exe" ]; then
        echo "Error: ${binary} not found after extraction"
        exit 1
    fi
}

# Detect Helm version
helm_major_version=""
if command -v helm > /dev/null 2>&1; then
    helm_version_output=$(helm version --short 2>/dev/null || echo "")
    helm_major_version=$(echo "$helm_version_output" | grep -oE 'v[0-9]+' | head -1 | tr -d 'v')
fi

# Helm 4: install the two sub-plugins directly, then disable the root plugin
if [ "$helm_major_version" = "4" ]; then
    echo "Helm 4 detected -- installing sub-plugins for full cli + getter support..."

    plugins_dir="$(dirname "$HELM_PLUGIN_DIR")"

    for entry in gcs:helm-gcs:helm-gcs-plugin gcs-getter:helm-gcs-getter:helm-gcs-getter-plugin; do
        src_dir="${entry%%:*}"                          # gcs or gcs-getter
        rest="${entry#*:}"
        binary="${rest%%:*}"                            # helm-gcs or helm-gcs-getter
        dest_name="${rest#*:}"                          # helm-gcs-plugin or helm-gcs-getter-plugin

        src="$HELM_PLUGIN_DIR/plugins/${src_dir}"
        dest="${plugins_dir}/${dest_name}"

        if [ ! -d "$src" ]; then
            echo "Error: sub-plugin source not found: $src"
            exit 1
        fi

        rm -rf "$dest"
        cp -r "$src" "$dest"
        download_binary "$binary" "$dest/bin"
    done

    # Disable the root plugin so Helm 4 doesn't see a duplicate "gcs" plugin.
    # We rename rather than delete in case this is a local dev checkout.
    mv -f "$HELM_PLUGIN_DIR/plugin.yaml" "$HELM_PLUGIN_DIR/plugin.yaml.bak" 2>/dev/null || true

    echo ""
    echo "helm-gcs ${version} installed for Helm 4."
    echo ""
    echo "  helm gcs init gs://bucket/path         # Initialize repository"
    echo "  helm repo add myrepo gs://bucket/path   # Add repository"
    echo "  helm gcs push chart.tgz myrepo          # Push a chart"
    echo "  helm gcs rm chart myrepo                # Remove a chart"
    echo ""
    exit 0
fi

# Helm 3: legacy single-plugin install (both binaries into this plugin's bin/)
echo "Installing helm-gcs ${version} ..."

rm -rf bin
mkdir -p bin

for binary in helm-gcs helm-gcs-getter; do
    download_binary "$binary" bin
done

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
