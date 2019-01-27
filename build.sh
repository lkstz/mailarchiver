#!/usr/bin/env bash

BUILD_DIR=build
PROGRAM_NAME=mailarchiver

# Check for version argument
if [[ -z "$1" ]]; then
	echo "Please supply a version string"
	exit 1
fi

# Ensure that working dir is clean
if [[ ! -z "$(git status --porcelain)" ]]; then
	echo "Unstaged changes exist, please commit all changes"
	exit 1
fi


# Replace version in go file
sed -i -E "s/version = \"[0-9]+\.[0-9]+\.[0-9]+\"/version = \"$1\"/g" mailarchiver.go

# Commit version bump
git add mailarchiver.go
git commit -m "Bump version $1"
git tag v$1


# Build binaries
for GOOS in linux windows darwin; do
    for GOARCH in 386 amd64; do
        go build -v -o ${BUILD_DIR}/${PROGRAM_NAME}-$1-${GOOS}_${GOARCH}
    done
done
