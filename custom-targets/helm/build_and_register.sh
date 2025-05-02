#!/bin/bash

SOURCE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

export _CT_SRCDIR="${SOURCE_DIR}/helm-deployer"
export _CT_IMAGE_NAME=helm
export _CT_TYPE_NAME=helm
export _CT_CUSTOM_ACTION_NAME=helm-deployer
export _CT_GCS_DIRECTORY=helm
export _CT_SKAFFOLD_CONFIG_NAME=helmConfig

"${SOURCE_DIR}/../util/build_and_register.sh" "$@"