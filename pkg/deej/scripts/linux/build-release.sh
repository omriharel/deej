#!/bin/sh

echo 'Building deej (release)...'

# shove git commit, version tag into env
GIT_COMMIT=$(git rev-list -1 --abbrev-commit HEAD)
VERSION_TAG=$(git describe --tags --always)
BUILD_TYPE=release
echo 'Embedding build-time parameters:'
echo "- gitCommit $GIT_COMMIT"
echo "- versionTag $VERSION_TAG"
echo "- buildType $BUILD_TYPE"

go build -o deej-release -ldflags "-s -w -X main.gitCommit=$GIT_COMMIT -X main.versionTag=$VERSION_TAG -X main.buildType=$BUILD_TYPE" ./pkg/deej/cmd
if [ $? -eq 0 ]; then
    echo 'Done.'
else
    echo 'Error: "go build" exited with a non-zero code. Are you running this script from the root deej directory?'
    exit 1
fi

