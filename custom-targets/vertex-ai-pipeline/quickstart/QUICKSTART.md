# Cloud Deploy Vertex AI Model Deployer Quickstart

## Overview

This quickstart demonstrates how to deploy a ML Pipeline to an target environment using a Cloud Deploy custom target.


## 1. Clone Repository

Clone this repository and navigate to the quickstart directory (`cloud-deploy-samples/custom-targets/vertex-ai-pipeline/quickstart`) since the commands provided expect to be executed from that directory.

## 2. Environment variables

To simplify the commands in this quickstart, set the following environment variables with your values:

```shell
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="YOUR_REGION"
export BUCKET_NAME="YOUR_BUCKET"
export BUCKET_URI="gs://$BUCKET_NAME"
```

```shell
export PROJECT_ID="scortabarria-internship"
export REGION="us-central1"
export BUCKET_NAME="pipeline-artifacts-scorta"
export BUCKET_URI="gs://$BUCKET_NAME"
```
## 3. Prerequisites

[Install](https://cloud.google.com/sdk/docs/install) the latest version of the Google Cloud CLI


### APIs
Enable the Cloud Deploy API, Compute Engine API, and Vertex AI API.
   ```shell
   gcloud services enable clouddeploy.googleapis.com aiplatform.googleapis.com compute.googleapis.com --project $PROJECT_ID
   ```
### Permissions
The default service account, `{project_num}-compute@developer.gserviceaccount.com`, used by Cloud Deploy needs the
   following roles:

1. `roles/clouddeploy.jobRunner` - required by Cloud Deploy

   ```shell
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
       --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
       --role="roles/clouddeploy.jobRunner"
   ```
2. `roles/clouddeploy.viewer` - required to access Cloud Deploy resources

   ```shell
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
       --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
       --role="roles/clouddeploy.viewer"
   ```
3. `roles/aiplatform.user` - required to access the models and deploy endpoints in the custom target

   ```shell
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
       --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
       --role="roles/aiplatform.user"
   ```

4. Create a bucket

```shell
gsutil mb -l $REGION -p $PROJECT_ID gs://$BUCKET_NAME
```


5. Build and Register a Custom Target Type for Vertex AI

From within the `quickstart` directory, run this command to build the Vertex AI model deployer image and
install the custom target resources:

```shell
../build_and_register.sh -p $PROJECT_ID -r $REGION
```

For information about the `build_and_register.sh` script, see the [README](../README.md#build)


## 6. Create delivery pipeline, target, and skaffold

Within the `quickstart` directory, run this second command to make a temporary copy of `clouddeploy.yaml` and
`configuration/skaffold.yaml`, and to replace placeholders in the copies with actual values

```shell
export TMPDIR=$(mktemp -d)
./replace_variables.sh -p $PROJECT_ID -r $REGION -e $ENDPOINT_ID -t $TMPDIR -b $BUCKET_NAME
```

The command does the following:
1. Creates temporary directory $TMPDIR and copies `clouddeploy.yaml` and `configuration` into it.
2. Replaces the placeholders in `$TMPDIR/clouddeploy.yaml`
3. Obtains the URL of the latest version of the custom image, built in step 6, and sets it in `$TMPDIR/configuration/skaffold.yaml`


Lastly, apply the Cloud Deploy configuration defined in `clouddeploy.yaml`:

```shell
gcloud deploy apply --file=$TMPDIR/clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

## 7. Create a release and rollout

Create a Cloud Deploy release for the configuration defined in the `configuration` directory. This automatically
creates a rollout that deploys the first model version to the target.

```shell
gcloud deploy releases create release-001 \
    --delivery-pipeline=pipeline-cd \
    --project=$PROJECT_ID \
    --region=$REGION \
    --source=$TMPDIR/configuration \
    --deploy-parameters="customTarget/vertexAIPipeline=https://us-central1-kfp.pkg.dev/scortabarria-internship/scortabarria-internship-rlhf-pipelines/rlhf-tune-pipeline/sha256:e739c5c310d406f8a6a9133b0c97bf9a249715da0a507505997ced042e3e0f17"
```

### Explanation of command line flags

The `--source` command line flag instructs gcloud where to look for the configuration files relative to the working directory where the command is run.

The `--deploy-parameters` flag is used to provide the custom deployer with additional parameters needed to perform the deployment.

Here, we are providing the custom deployer with deploy parameter `customTarget/vertexAIModel`
which specifies the full resource name of the model to deploy

The remaining flags specify the Cloud Deploy Delivery Pipeline. `--delivery-pipeline` is the name of
the delivery pipeline where the release will be created, and the project and region of the pipeline
is specified by `--project` and `--region` respectively.


### Monitor the release's progress

To check release details, run this command:

```shell
gcloud deploy releases describe release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION
```

Run this command to filter only the render status of the release:

```shell
gcloud deploy releases describe release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION --format "(renderState)"
```

## 8. Monitor rollout status

In the [Cloud Deploy UI](https://cloud.google.com/deploy) for your project click on the
`vertex-ai-cloud-deploy-pipeline` delivery pipeline. Here you can see the release created and the rollout to the target for the release.

You can also describe the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-001-to-prod-endpoint-0001 --release=release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION
```

It will take up to 15 minutes for the model to fully deploy.

After the rollout completes, you can inspect the deployed models and traffic splits of the endpoint with `gcloud`

```shell
gcloud ai endpoints describe $ENDPOINT_ID --region $REGION --project $PROJECT_ID
```
## 9. Inspect aliases in the deployed model 

Monitor the post-deploy operation by querying the rollout:

```
gcloud deploy rollouts describe release-001-to-prod-endpoint-0001 --release=release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION --format "(phases[0].deploymentJobs.postdeployJob)"
```

After the post-deploy job has succeeded, you can then inspect the deployed model and view its currently assigned aliases. `prod` and `champion` should be assigned.

```shell
gcloud ai models describe test_model --region $REGION --project $PROJECT_ID --format "(versionAliases)"
```

## 10. Clean up

To delete the endpoint after the quickstart, run the following commands:

Obtain the id of the deployed model:
```shell
gcloud ai endpoints describe $ENDPOINT_ID --region $REGION --project $PROJECT_ID --format "(deployedModels[0].id)"
```

Undeploy the model:
```shell
gcloud ai endpoints undeploy-model $ENDPOINT_ID --region $REGION --project $PROJECT_ID --deployed-model-id $DEPLOYED_MODEL_ID
```

Delete the endpoint:
```shell
gcloud ai endpoints delete $ENDPOINT_ID --region $REGION --project $PROJECT_ID
```

To delete the imported model:

```shell
gcloud ai models delete test_model --region $REGION --project $PROJECT_ID
```

To delete Cloud Deploy resources:

```shell
gcloud deploy delete --file=$TMPDIR/clouddeploy.yaml --force --project=$PROJECT_ID --region=$REGION
```
