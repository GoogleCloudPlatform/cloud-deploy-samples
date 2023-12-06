#!/bin/bash

set -e

if [[ ! -v _CT_SRCDIR || ! -v _CT_IMAGE_NAME || ! -v _CT_TYPE_NAME || ! -v _CT_CUSTOM_ACTION_NAME || ! -v _CT_GCS_DIRECTORY || ! -v _CT_SKAFFOLD_CONFIG_NAME ]]; then
  echo "This script is not meant to be used on its own. Please launch it from one of the custom target directories."
  exit 1
fi

usage() {
  echo "usage: build_and_register.sh -p <project_id> -r <region>"
}

boldout() {
  echo $(tput bold)$(tput setaf 1)"$@"$(tput sgr0)
}

while getopts "p:r:" arg; do
  case "${arg}" in
    p)
      PROJECT="${OPTARG}"
      ;;
    r)
      REGION="${OPTARG}"
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

if [[ ! -v PROJECT || ! -v REGION ]]; then
  usage
  exit 1
fi

AR_REPO=$REGION-docker.pkg.dev/$PROJECT/cd-custom-targets
if ! gcloud -q artifacts repositories describe --location "$REGION" --project "$PROJECT" cd-custom-targets > /dev/null 2>&1; then
  boldout "Creating Artifact Registry repository: ${AR_REPO}"
  gcloud -q artifacts repositories create --location "$REGION" --project "$PROJECT" --repository-format docker cd-custom-targets
fi

boldout "Granting the default compute service account access to ${AR_REPO}"
gcloud -q artifacts repositories add-iam-policy-binding \
    --project "${PROJECT}" --location "${REGION}" cd-custom-targets \
    --member=serviceAccount:$(gcloud -q projects describe $PROJECT --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.reader" > /dev/null

BUCKET_NAME="${PROJECT}-${REGION}-custom-targets"
if ! gsutil ls "gs://${BUCKET_NAME}" > /dev/null 2>&1; then
  boldout "Creating a storage bucket to hold the custom target configuration"
  gcloud -q storage buckets create --project "${PROJECT}" --location "${REGION}" "gs://${BUCKET_NAME}"
fi

boldout "Building the Custom Target image in Cloud Build."
boldout "This will take approximately 10 minutes"

# get the commit hash to pass to the build
COMMIT_SHA=$(git rev-parse --verify HEAD)

CLOUDBUILD_YAML="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )/cloudbuild.yaml"
# TODO(plumpy): just pass the URL to the cloudbuild.yaml file in Github, once it's public
# Using `beta` because the non-beta command won't stream the build logs
gcloud -q beta builds submit --project="$PROJECT" --region="$REGION" \
    --substitutions=_AR_REPO_NAME=cd-custom-targets,_IMAGE_NAME=${_CT_IMAGE_NAME},COMMIT_SHA="${COMMIT_SHA}" \
    --config="${CLOUDBUILD_YAML}" \
    "${_CT_SRCDIR}"

IMAGE_SHA=$(gcloud -q artifacts docker images describe "${AR_REPO}/${_CT_IMAGE_NAME}:latest" --format 'get(image_summary.digest)')

TMPDIR=$(mktemp -d)
trap 'rm -rf -- "${TMPDIR}"' EXIT

boldout "Uploading the custom target definition to gs://${BUCKET_NAME}"
cat >"${TMPDIR}/skaffold.yaml" <<EOF
apiVersion: skaffold/v4beta7
kind: Config
metadata:
  name: ${_CT_SKAFFOLD_CONFIG_NAME}
customActions:
  - name: ${_CT_CUSTOM_ACTION_NAME}
    containers:
      - name: ${_CT_CUSTOM_ACTION_NAME}
        image: ${AR_REPO}/${_CT_IMAGE_NAME}@${IMAGE_SHA}
EOF
gsutil -q cp "${TMPDIR}/skaffold.yaml" "gs://${BUCKET_NAME}/${_CT_GCS_DIRECTORY}/skaffold.yaml"

boldout "Create the CustomTargetType resource in Cloud Deploy"
cat >"${TMPDIR}/clouddeploy.yaml" <<EOF
apiVersion: deploy.cloud.google.com/v1
kind: CustomTargetType
metadata:
  name: ${_CT_TYPE_NAME}
customActions:
EOF
if [[ ! -v _CT_USE_DEFAULT_RENDERER ]]; then
  echo "  renderAction: ${_CT_CUSTOM_ACTION_NAME}" >> "${TMPDIR}/clouddeploy.yaml"
fi
cat >>"${TMPDIR}/clouddeploy.yaml" <<EOF
  deployAction: ${_CT_CUSTOM_ACTION_NAME}
  includeSkaffoldModules:
    - configs: ["${_CT_SKAFFOLD_CONFIG_NAME}"]
      googleCloudStorage:
        source: "gs://${BUCKET_NAME}/${_CT_GCS_DIRECTORY}/*"
        path: "skaffold.yaml"
EOF
gcloud -q deploy apply --project "${PROJECT}" --region "${REGION}" --file "${TMPDIR}/clouddeploy.yaml"
