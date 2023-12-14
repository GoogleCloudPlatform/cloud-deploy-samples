#!/bin/bash

SOURCE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

export _CT_SRCDIR="${SOURCE_DIR}/im-deployer"
export _CT_IMAGE_NAME=infra-manager
export _CT_TYPE_NAME=infrastructure-manager
export _CT_CUSTOM_ACTION_NAME=infra-manager-deployer
export _CT_GCS_DIRECTORY=infra-manager
export _CT_SKAFFOLD_CONFIG_NAME=infraManagerConfig

"${SOURCE_DIR}/../util/build_and_register.sh" "$@"
