#!/bin/sh

version=`grep version plugin.yaml | cut -d '"' -f 2`
git tag $version && git push --tags
goreleaser
