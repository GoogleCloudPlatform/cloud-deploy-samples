#!/bin/bash

DEV_CLUSTER=projects/<projectId>/locations/<location>/clusters/<clusterName>
STAGING_CLUSTER=projects/<projectId>/locations/<location>/clusters/<clusterName>
PROD1_CLUSTER=projects/<projectId>/locations/<location>/clusters/<clusterName>
PROD2_CLUSTER=projects/<projectId>/locations/<location>/clusters/<clusterName>
COMPUTE_SERVICE_ACCOUNT=<projectNumber>-compute@developer.gserviceaccount.com
IMAGE_REPO="Repo for images" # Ex: us-central1-docker.pkg.dev/<project>/<repoName>
GIT_REPO=https://github.com/GoogleCloudPlatform/cloud-deploy-samples

for file in clouddeploy.yaml \
colors-fd/cloudbuild.yaml colors-fd/k8s.yaml colors-fd/skaffold.yaml \
colors-be/cloudbuild.yaml colors-be/k8s.yaml colors-be/skaffold.yaml; do
    echo Updating $file
    sed 's,\$DEV_CLUSTER,'$DEV_CLUSTER',' $file.template | \
    sed 's,\$STAGING_CLUSTER,'$STAGING_CLUSTER',' | \
    sed 's,\$PROD1_CLUSTER,'$PROD1_CLUSTER',' | \
    sed 's,\$COMPUTE_SERVICE_ACCOUNT,'$COMPUTE_SERVICE_ACCOUNT',' | \
    sed 's,\$IMAGE_REPO,'$IMAGE_REPO',' | \
    sed 's,\$PROD2_CLUSTER,'$PROD2_CLUSTER',' > $file
done