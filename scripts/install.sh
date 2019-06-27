#!/bin/sh

cd $HELM_PLUGIN_DIR
version="$(cat plugin.yaml | grep "version" | cut -d '"' -f 2)"
echo "Installing helm-gcs ${version} ..."

# Find correct archive name
unameOut="$(uname -s)"

case "${unameOut}" in
    Linux*)             os=Linux;;
    Darwin*)            os=Darwin;;
    CYGWIN*)            os=Cygwin;;
    MINGW*|MSYS_NT*)    os=windows;;
    *)                  os="UNKNOWN:${unameOut}"
esac

arch=`uname -m`

if echo "$os" | grep -qe '.*UNKNOWN.*'
then
    echo "Unsupported OS / architecture: ${os}_${arch}"
    exit 1
fi

url="https://github.com/hayorov/helm-gcs/releases/download/${version}/helm-gcs_${version}_${os}_${arch}.tar.gz"

filename=`echo ${url} | sed -e "s/^.*\///g"`

# Download archive
if [ -n $(command -v curl) ]
then
    curl -sSL -O $url
elif [ -n $(command -v wget) ]
then
    wget -q $url
else
    echo "Need curl or wget"
    exit -1
fi

# Install bin
rm -rf bin && mkdir bin && tar xzvf $filename -C bin > /dev/null && rm -f $filename

echo "helm-gcs ${version} is correctly installed."
echo

echo "Init a new repository:"
echo "  helm gcs init gs://bucket/path"
echo

echo "Add your repository to Helm:"
echo "  helm repo add repo-name gs://bucket/path"
echo

echo "Push a chart to your repository:"
echo "  helm gcs push chart.tar.gz repo-name"
echo

echo "Update Helm cache:"
echo "  helm repo update"
echo

echo "Get your chart:"
echo "  helm fetch repo-name/chart"
echo
