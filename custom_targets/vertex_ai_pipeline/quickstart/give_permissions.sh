#!/bin/bash

# PERMISSION FOR PIPELINE PROJECT
gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
       --member=serviceAccount:${PIPELINE_PROJECT_NUMBER}-compute@developer.gserviceaccount.com \
       --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/logging.logWriter"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.viewer"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/storage.objectUser"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/aiplatform.user"


gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $PIPELINE_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"



# PERMISSIONS FOR TARGET PROJECT
gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.viewer"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/aiplatform.user"



gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.viewer"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PIPELINE_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/aiplatform.user"





gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.viewer"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/editor"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $STAGING_PROJECT_ID \
    --member=serviceAccount:$STAGING_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/aiplatform.user"



gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.viewer"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/editor"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $PROD_PROJECT_ID \
    --member=serviceAccount:$PROD_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
    --role="roles/aiplatform.user"