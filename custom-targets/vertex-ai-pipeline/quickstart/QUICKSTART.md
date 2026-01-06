# Cloud Deploy Vertex AI Pipeline Deployer Quickstart

## Overview

This quickstart demonstrates how to deploy a ML Pipeline to two target environments using Cloud Deploy custom targets.

## 0. Prior setup

In this quickstart, we will go over how to deploy an ML Pipeline to a staging and then a production environment. This means that you must have three projects: the first to hold your pipeline template (a.k.a. PIPELINE_PROJECT), the second as a staging environment (a.k.a. STAGING_PROJECT), and finally the third as a production environment (a.k.a. PROD_PROJECT). In order for this quickstart to be successfull, you must also have the following things defined:
1. A pipeline template in PIPELINE_PROJECT. The REPOSITORY_ID, PACKAGE_ID, and TAG/VERSION of this pipeline template will be used later to identify it.
2. Preference and prompt datasets in your STAGING_PROJECT. These will be used to run your pipeline in its staging environment.
3. Preference and prompt datasets in your PROD_PROJECT. These will be used to run your pipeline in its production environment.


## 1. Clone repository

Clone this repository and navigate to the quickstart directory (`cloud-deploy-samples/custom-targets/vertex-ai-pipeline/quickstart`) since the commands provided expect to be executed from that directory.


## 2. Environment variables

To simplify the commands in this quickstart, set the following environment variables with your values:

```shell
export PIPELINE_PROJECT_ID="YOUR_PIPELINE_PROJECT_ID"
export PIPELINE_REGION="YOUR_PIPELINE_REGION"
export PIPELINE_PROJECT_NUMBER=$(gcloud projects list \
        --format="value(projectNumber)" \
        --filter="projectId=${PIPELINE_PROJECT_ID}")

export STAGING_PROJECT_ID="YOUR_STAGING_PROJECT_ID"
export STAGING_REGION="YOUR_STAGING_REGION"
export STAGING_PROJECT_NUMBER=$(gcloud projects list \
        --format="value(projectNumber)" \
        --filter="projectId=${STAGING_PROJECT_ID}")
export STAGING_BUCKET_NAME="GIVE_YOUR_STAGING_BUCKET_A_NAME"
export STAGING_PREF_DATA="YOUR_STAGING_PREFERENCE_DATASET"
export STAGING_PROMPT_DATA="YOUR_STAGING_PROMPT_DATASET"

export PROD_PROJECT_ID="YOUR_PROD_PROJECT_ID"
export PROD_REGION="YOUR_PROD_REGION"
export PROD_PROJECT_NUMBER=$(gcloud projects list \
        --format="value(projectNumber)" \
        --filter="projectId=${PROD_PROJECT_ID}")
export PROD_BUCKET_NAME="GIVE_YOUR_PROD_BUCKET_A_NAME"
export PROD_PREF_DATA="YOUR_PROD_PREFERENCE_DATASET"
export PROD_PROMPT_DATA="YOUR_PROD_PROMPT_DATASET"

export REPO_ID="YOUR_REPO"
export PACKAGE_ID="YOUR_PACKAGE"
export TAG_OR_VERSION="YOUR_TAG_OR_VERSION"
export LARGE_MODEL_REFERENCE="YOUR_LARGE_MODEL_REFERENCE"
export MODEL_DISPLAY_NAME="YOUR_DISPLAY_NAME"
```


## 3. Prerequisites

[Install](https://cloud.google.com/sdk/docs/install) the latest version of the Google Cloud CLI


### APIs
Enable the Cloud Deploy API, Compute Engine API, Artifact Registry API and Vertex AI API for the project where your pipeline template is located. For each target, enable the Vertex AI API.

   ```shell
   gcloud services enable clouddeploy.googleapis.com aiplatform.googleapis.com compute.googleapis.com artifactregistry.googleapis.com cloudbuild.googleapis.com --project $PIPELINE_PROJECT_ID
   ```

   ```shell
   gcloud services enable aiplatform.googleapis.com --project $STAGING_PROJECT_ID
   ```

   ```shell
   gcloud services enable aiplatform.googleapis.com --project $PROD_PROJECT_ID
   ```


## 4. Create a Bucket

From the `quickstart` directory, run these commands to create a bucket in Cloud Storage for each of your targets:

```shell
gcloud storage buckets create gs://$STAGING_BUCKET_NAME --location $STAGING_REGION --project $STAGING_PROJECT_ID

gcloud storage buckets create gs://$PROD_BUCKET_NAME --location $PROD_REGION --project $PROD_PROJECT_ID

```


## 5. Build and Register a Custom Target Type for Vertex AI

From within the `quickstart` directory, run this command to build the Vertex AI model deployer image and
install the custom target resources:

```shell
../build_and_register.sh -p $PIPELINE_PROJECT_ID -r $PIPELINE_REGION
```

For information about the `build_and_register.sh` script, see the [README](../README.md#build)


## 6. Create delivery pipeline, target, and skaffold

Within the `quickstart` directory, run this second command to make a temporary copy of `clouddeploy.yaml`, `configuration/skaffold.yaml` and
`configuration/staging/pipelineJob.yaml`, and to replace placeholders in the copies with actual values:

```shell
export TMPDIR=$(mktemp -d)
./replace_variables.sh -s $STAGING_PROJECT_ID -r $STAGING_REGION -p $PROD_PROJECT_ID -o $PROD_REGION -t $TMPDIR -b $STAGING_BUCKET_NAME -c $PROD_BUCKET_NAME -f $STAGING_PREF_DATA -m $STAGING_PROMPT_DATA -l $LARGE_MODEL_REFERENCE -d $MODEL_DISPLAY_NAME -y $PROD_PREF_DATA -z $PROD_PROMPT_DATA -e $STAGING_PROJECT_NUMBER -g $PROD_PROJECT_NUMBER -h $PIPELINE_PROJECT_NUMBER -i $PIPELINE_PROJECT_ID -j $PIPELINE_REGION
```

The command does the following:
1. Creates temporary directory $TMPDIR and copies `clouddeploy.yaml`, `give_permissions.sh`, and `configuration` into it.
2. Replaces the placeholders in `$TMPDIR/clouddeploy.yaml`, `configuration/skaffold.yaml`, `give_permissions.sh`, `configuration/staging/pipelineJob.yaml`, and `configuration/production/pipelineJob.yaml`
3. Obtains the URL of the latest version of the custom image, built in step 6, and sets it in `$TMPDIR/configuration/skaffold.yaml`


### Permissions
The default service account, `{project_num}-compute@developer.gserviceaccount.com`, used by Cloud Deploy needs a few permissions. Run this command to give the necessary permissions to your service accounts:

```shell
./give_permissions.sh
```


### Create a delivery pipeline
Lastly, apply the Cloud Deploy configuration defined in `clouddeploy.yaml`:

```shell
gcloud deploy apply --file=$TMPDIR/clouddeploy.yaml --project=$PIPELINE_PROJECT_ID --region=$PIPELINE_REGION
```


## 7. Create a release and rollout

Create a Cloud Deploy release for the configuration defined in the `configuration` directory. This automatically
creates a rollout that deploys the pipeline to the staging environment.

```shell
gcloud deploy releases create release-001 \
    --delivery-pipeline=pipeline-cd \
    --project=$PIPELINE_PROJECT_ID \
    --region=$PIPELINE_REGION \
    --source=$TMPDIR/configuration \
    --deploy-parameters="customTarget/vertexAIPipeline=https://$PIPELINE_REGION-kfp.pkg.dev/$PIPELINE_PROJECT_ID/$REPO_ID/$PACKAGE_ID/$TAG_OR_VERSION"
```


### Explanation of command line flags

The `--source` command line flag instructs gcloud where to look for the configuration files relative to the working directory where the command is run.

The `--deploy-parameters` flag is used to provide the custom deployer with additional parameters needed to perform the deployment.

Here, we are providing the custom deployer with deploy parameter `customTarget/vertexAIPipeline`
which specifies the full resource name of the pipeline to deploy

The remaining flags specify the Cloud Deploy Delivery Pipeline. `--delivery-pipeline` is the name of
the delivery pipeline where the release will be created, and the project and region of the pipeline
is specified by `--project` and `--region` respectively.


### Monitor the release's progress

To check release details, run this command:

```shell
gcloud deploy releases describe release-001 --delivery-pipeline=pipeline-cd --project=$PIPELINE_PROJECT_ID --region=$PIPELINE_REGION
```

Run this command to filter only the render status of the release:

```shell
gcloud deploy releases describe release-001 --delivery-pipeline=pipeline-cd --project=$PIPELINE_PROJECT_ID --region=$PIPELINE_REGION --format "(renderState)"
```


## 8. Monitor rollout status

In the [Cloud Deploy UI](https://cloud.google.com/deploy) for your project click on the
`pipeline-cd` delivery pipeline. Here you can see the release created and the rollout to the target for the release.

You can also describe the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-001-to-staging-environment-0001 --release=release-001 --delivery-pipeline=pipeline-cd --project=$PIPELINE_PROJECT_ID --region=$PIPELINE_REGION
```

 
## 9. Promote a release

This promotes the release, automatically moving it to the production environment.

```shell
gcloud deploy releases promote \
    --release=release-001 \
    --delivery-pipeline=pipeline-cd \
    --to-target=prod-environment \
    --project=$PIPELINE_PROJECT_ID \
    --region=$PIPELINE_REGION
```

You will be asked to confirm whether you would like to complete this promotion. Entering `y` or `Y` will finish the action, deploying the pipeline to the production environment. 


### Monitor the release's progress

To check release details, run this command:

```shell
gcloud deploy releases describe release-001 --delivery-pipeline=pipeline-cd --project=$PROD_PROJECT_ID --region=$PROD_REGION
```

Run this command to filter only the render status of the release:

```shell
gcloud deploy releases describe release-001 --delivery-pipeline=pipeline-cd --project=$PROD_PROJECT_ID --region=$PROD_REGION --format "(renderState)"
```


## 10. Monitor rollout status

In the [Cloud Deploy UI](https://cloud.google.com/deploy) for your project click on the
`pipeline-cd` delivery pipeline. Here you can see the release created and the rollout to the target for the release.

You can also describe the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-001-to-staging-environment-0001 --release=release-001 --delivery-pipeline=pipeline-cd --project=$PROD_PROJECT_ID --region=$PROD_REGION
```


## 11. Clean up

You have now completed the quickstart, deploying an ML Pipeline to two target environments. To delete the Cloud Deploy resources:

```shell
gcloud deploy delete --file=$TMPDIR/clouddeploy.yaml --force --project=$PIPELINE_PROJECT_ID --region=$PIPELINE_REGION
```
