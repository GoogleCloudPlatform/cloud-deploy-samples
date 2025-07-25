#!/bin/bash

# Get the name of the directory where this script is located.
SOURCE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PARENT_DIR="$(cd "$SOURCE_DIR/../../" && pwd)"

# TODO: b/430551407 - Remove _CT_SRCDIR once the refactor is complete.
export _CT_SRCDIR="${PARENT_DIR}/"
export _CT_DOCKERFILE_LOCATION="custom-targets/git-ops/git-deployer/Dockerfile"
export _CT_IMAGE_NAME=git
export _CT_TYPE_NAME=git
export _CT_CUSTOM_ACTION_NAME=git-deployer
export _CT_GCS_DIRECTORY=git
export _CT_SKAFFOLD_CONFIG_NAME=gitConfig
export _CT_USE_DEFAULT_RENDERER=true

"${SOURCE_DIR}/../util/build_and_register.sh" "$@"