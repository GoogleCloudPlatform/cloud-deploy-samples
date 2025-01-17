#!/bin/bash

export _CT_IMAGE_NAME=vertexai

while getopts "p:r:d:e:t:" arg; do
  case "${arg}" in
    p)
      PROJECT="${OPTARG}"
      ;;
    r)
      REGION="${OPTARG}"
      ;;
    d)
      DEV_ENDPOINT="${OPTARG}"
      ;;
    e)
      PROD_ENDPOINT="${OPTARG}"
      ;;
    t)
      TMPDIR="${OPTARG}"
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

if [[ ! -v PROJECT || ! -v REGION || ! -v DEV_ENDPOINT || ! -v PROD_ENDPOINT || ! -v TMPDIR ]]; then
  usage
  exit 1
fi

# get the location where the custom image was uploaded
AR_REPO=$REGION-docker.pkg.dev/$PROJECT/cd-custom-targets

# get the image digest of hte most recently built image
IMAGE_SHA=$(gcloud -q artifacts docker images describe "${AR_REPO}/${_CT_IMAGE_NAME}:latest" --format 'get(image_summary.digest)')

# make a temporary copy of clouddeploy.yaml and skaffold.yaml
cp clouddeploy.yaml "$TMPDIR"/clouddeploy.yaml
cp -r configuration "$TMPDIR"/configuration

# replace variables in clouddeploy.yaml with actual values
sed -i "s/\$PROJECT_ID/${PROJECT}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$REGION/${REGION}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$DEV_ENDPOINT_ID/${DEV_ENDPOINT}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$PROD_ENDPOINT_ID/${PROD_ENDPOINT}/g" "$TMPDIR"/clouddeploy.yaml

# replace variables in configuration/skaffold.yaml with actual values
sed -i "s/\$REGION/${REGION}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$PROJECT_ID/${PROJECT}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$_CT_IMAGE_NAME/${_CT_IMAGE_NAME}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$IMAGE_SHA/${IMAGE_SHA}/g" "$TMPDIR"/configuration/skaffold.yaml


