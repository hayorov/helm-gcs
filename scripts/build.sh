#!/bin/sh

version=`grep version plugin.yaml | cut -d '"' -f 2`
echo "version:" $version
git tag $version && git push --tags
goreleaser
