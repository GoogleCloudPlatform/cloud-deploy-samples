#!/bin/bash

export _CT_DOCKERFILE_LOCATION="custom-targets/vertex-ai-pipeline/pipeline-deployer/Dockerfile"
export _CT_IMAGE_NAME=vertexai
export _CT_TYPE_NAME=vertex-ai-pipeline
export _CT_CUSTOM_ACTION_NAME=vertex-ai-pipeline-deployer
export _CT_GCS_DIRECTORY=vertexai
export _CT_SKAFFOLD_CONFIG_NAME=vertexAiConfig

"${SOURCE_DIR}/../util/build_and_register.sh" "$@"

