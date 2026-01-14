#!/bin/bash

SOURCE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

export _CT_DOCKERFILE_LOCATION="custom-targets/terraform/terraform-deployer/Dockerfile"
export _CT_IMAGE_NAME=terraform
export _CT_TYPE_NAME=terraform
export _CT_CUSTOM_ACTION_NAME=terraform-deployer
export _CT_GCS_DIRECTORY=terraform
export _CT_SKAFFOLD_CONFIG_NAME=terraformConfig

"${SOURCE_DIR}/../util/build_and_register.sh" "$@"
