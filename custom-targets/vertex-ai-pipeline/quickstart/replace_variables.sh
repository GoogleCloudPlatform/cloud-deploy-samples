#!/bin/bash

export _CT_IMAGE_NAME=vertexai

while getopts "p:r:e:t:b:" arg; do
  case "${arg}" in
    p)
      PROJECT="${OPTARG}"
      ;;
    r)
      REGION="${OPTARG}"
      ;;
    e)
      ENDPOINT="${OPTARG}"
      ;;
    t)
      TMPDIR="${OPTARG}"
      ;;
    b)
      BUCKET_NAME="${OPTARG}"
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

if [[ ! -v PROJECT || ! -v REGION || ! -v ENDPOINT || ! -v TMPDIR || ! -v BUCKET_NAME ]]; then
  usage
  exit 1
fi

# get the location where the custom image was uploaded
AR_REPO=$REGION-docker.pkg.dev/$PROJECT/cd-custom-targets

# get the image digest of the most recently built image
IMAGE_SHA=$(gcloud -q artifacts docker images describe "${AR_REPO}/${_CT_IMAGE_NAME}:latest" --format 'get(image_summary.digest)')


cp clouddeploy.yaml "$TMPDIR"/clouddeploy.yaml
cp -r configuration "$TMPDIR"/configuration

# replace variables in clouddeploy.yaml with actual values
sed -i "s/\$PROJECT_ID/${PROJECT}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$REGION/${REGION}/g" "$TMPDIR"/clouddeploy.yaml
# sed -i "s/\$ENDPOINT_ID/${ENDPOINT}/g" "$TMPDIR"/clouddeploy.yaml 

# replace variables in configuration/skaffold.yaml with actual values
sed -i "s/\$REGION/${REGION}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$PROJECT_ID/${PROJECT}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$_CT_IMAGE_NAME/${_CT_IMAGE_NAME}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$IMAGE_SHA/${IMAGE_SHA}/g" "$TMPDIR"/configuration/skaffold.yaml

# replace variables in configuration/staging/pipelineJob.yaml
sed -i "s/\$BUCKET_NAME/${BUCKET_NAME}/g" "$TMPDIR"/configuration/staging/pipelineJob.yaml



