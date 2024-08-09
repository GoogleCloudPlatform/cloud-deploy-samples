#!/bin/bash

export _CT_IMAGE_NAME=vertexai

while getopts "s:r:p:o:t:b:c:f:m:y:z:l:d:e:g:h:i:j:" arg; do
  case "${arg}" in
    s)
      STAGING_PROJECT="${OPTARG}"
      ;;
    r)
      STAGING_REGION="${OPTARG}"
      ;;
    p)
      PROD_PROJECT="${OPTARG}"
      ;;
    o)
      PROD_REGION="${OPTARG}"
      ;;
    t)
      TMPDIR="${OPTARG}"
      ;;
    b)
      STAGING_BUCKET="${OPTARG}"
      ;;
    c)
      PROD_BUCKET="${OPTARG}"
      ;;
    f)
      STAGING_PREF="${OPTARG}"
      ;;
    m)
      STAGING_PROMPT="${OPTARG}"
      ;;
    y)
      PROD_PREF="${OPTARG}"
      ;;
    z)
      PROD_PROMPT="${OPTARG}"
      ;;
    l)
      MODEL_REFERENCE="${OPTARG}"
      ;;
    d)
      DISPLAY="${OPTARG}"
      ;;
    e)
      STAGING_PROJECT_NUMBER="${OPTARG}"
      ;;
    g)
      PROD_PROJECT_NUMBER="${OPTARG}"
      ;;
    h)
      PIPELINE_PROJECT_NUMBER="${OPTARG}"
      ;;
    i)
      PIPELINE_PROJECT="${OPTARG}"
      ;;
    j)
      PIPELINE_REGION="${OPTARG}"
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

if [[ ! -v STAGING_PROJECT || ! -v STAGING_REGION || ! -v STAGING_PROJECT_NUMBER || ! -v PROD_PROJECT || ! -v PROD_REGION || ! -v PROD_PROJECT_NUMBER || ! -v TMPDIR || ! -v STAGING_BUCKET || ! -v PROD_BUCKET || ! -v STAGING_PREF || ! -v STAGING_PROMPT || ! -v PROD_PREF || ! -v PROD_PROMPT || ! -v MODEL_REFERENCE || ! -v DISPLAY || ! -v STAGING_PROJECT_NUMBER || ! -v PROD_PROJECT_NUMBER || ! -v PIPELINE_PROJECT_NUMBER || ! -v PIPELINE_PROJECT || ! -v PIPELINE_REGION ]]; then
  usage
  exit 1
fi

# get the location where the custom image was uploaded
AR_REPO=${PIPELINE_REGION}-docker.pkg.dev/${PIPELINE_PROJECT}/cd-custom-targets

# get the image digest of the most recently built image
IMAGE_SHA=$(gcloud -q artifacts docker images describe "${AR_REPO}/${_CT_IMAGE_NAME}:latest" --format 'get(image_summary.digest)')


cp clouddeploy.yaml "$TMPDIR"/clouddeploy.yaml
cp give_permissions.sh "$TMPDIR"/give_permissions.sh
cp -r configuration "$TMPDIR"/configuration


# replace variables in clouddeploy.yaml with actual values
sed -i "s/\$STAGING_PROJECT_ID/${STAGING_PROJECT}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$STAGING_REGION/${STAGING_REGION}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$PROD_PROJECT_ID/${PROD_PROJECT}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$PROD_REGION/${PROD_REGION}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s|\$STAGING_PREF_DATA|${STAGING_PREF}|g" "$TMPDIR"/clouddeploy.yaml
sed -i "s|\$STAGING_PROMPT_DATA|${STAGING_PROMPT}|g" "$TMPDIR"/clouddeploy.yaml
sed -i "s|\$PROD_PREF_DATA|${PROD_PREF}|g" "$TMPDIR"/clouddeploy.yaml
sed -i "s|\$PROD_PROMPT_DATA|${PROD_PROMPT}|g" "$TMPDIR"/clouddeploy.yaml
sed -i "s/\$LARGE_MODEL_REFERENCE/${MODEL_REFERENCE}/g" "$TMPDIR"/clouddeploy.yaml
sed -i "s|\$MODEL_DISPLAY_NAME|${DISPLAY}|g" "$TMPDIR"/clouddeploy.yaml


# # replace variables in configuration/skaffold.yaml with actual values
sed -i "s/\$STAGING_REGION/${STAGING_REGION}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$PROJECT_ID/${STAGING_PROJECT}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$_CT_IMAGE_NAME/${_CT_IMAGE_NAME}/g" "$TMPDIR"/configuration/skaffold.yaml
sed -i "s/\$IMAGE_SHA/${IMAGE_SHA}/g" "$TMPDIR"/configuration/skaffold.yaml

# replace variables in configuration/staging/pipelineJob.yaml and configuration/production/pipelineJob.yaml
sed -i "s|\$STAGING_BUCKET|${STAGING_BUCKET}|g" "$TMPDIR"/configuration/staging/pipelineJob.yaml
sed -i "s|\$PROD_BUCKET|${PROD_BUCKET}|g" "$TMPDIR"/configuration/production/pipelineJob.yaml
sed -i "s|\$STAGING_PROJECT_NUMBER|${STAGING_PROJECT_NUMBER}|g" "$TMPDIR"/configuration/staging/pipelineJob.yaml
sed -i "s|\$PROD_PROJECT_NUMBER|${PROD_PROJECT_NUMBER}|g" "$TMPDIR"/configuration/production/pipelineJob.yaml

# replace variables in configuration/staging/pipelineJob.yaml and configuration/production/pipelineJob.yaml
sed -i "s|\$STAGING_BUCKET|${STAGING_BUCKET}|g" "$TMPDIR"/configuration/staging/pipelineJob.yaml
sed -i "s|\$PROD_BUCKET|${PROD_BUCKET}|g" "$TMPDIR"/configuration/production/pipelineJob.yaml
sed -i "s|\$STAGING_PROJECT_NUMBER|${STAGING_PROJECT_NUMBER}|g" "$TMPDIR"/configuration/staging/pipelineJob.yaml
sed -i "s|\$PROD_PROJECT_NUMBER|${PROD_PROJECT_NUMBER}|g" "$TMPDIR"/configuration/production/pipelineJob.yaml

# replace variables in give_permissions.sh with actual values
sed -i "s|\$PIPELINE_PROJECT_ID|${PIPELINE_PROJECT}|g" "$TMPDIR"/give_permissions.sh
sed -i "s|\$PIPELINE_PROJECT_NUMBER|${PIPELINE_PROJECT_NUMBER}|g" "$TMPDIR"/give_permissions.sh
sed -i "s|\$STAGING_PROJECT_ID|${STAGING_PROJECT}|g" "$TMPDIR"/give_permissions.sh
sed -i "s|\$STAGING_PROJECT_NUMBER|${STAGING_PROJECT_NUMBER}|g" "$TMPDIR"/give_permissions.sh
sed -i "s|\$PROD_PROJECT_ID|${PROD_PROJECT}|g" "$TMPDIR"/give_permissions.sh
sed -i "s|\$PROD_PROJECT_NUMBER|${PROD_PROJECT_NUMBER}|g" "$TMPDIR"/give_permissions.sh
