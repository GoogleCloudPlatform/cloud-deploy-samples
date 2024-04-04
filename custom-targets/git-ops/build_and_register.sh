#!/bin/bash

SOURCE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

export _CT_SRCDIR="${SOURCE_DIR}/git-deployer"
export _CT_IMAGE_NAME=git
export _CT_TYPE_NAME=git
export _CT_CUSTOM_ACTION_NAME=git-deployer
export _CT_GCS_DIRECTORY=git
export _CT_SKAFFOLD_CONFIG_NAME=gitConfig
export _CT_USE_DEFAULT_RENDERER=true

"${SOURCE_DIR}/../util/build_and_register.sh" "$@"