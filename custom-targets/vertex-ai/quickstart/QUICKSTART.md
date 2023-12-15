# Cloud Deploy Vertex AI Model Deployer Quickstart

## Overview

This quickstart demonstrates how to deploy a Vertex AI model to an endpoint using a Cloud Deploy custom target.

In this quickstart you will:

1. Import a Vertex AI model into model registry and create an endpoint where the model will be deployed.
2. Define a Cloud Deploy delivery pipeline, custom target type for Vertex AI, and one target.
3. Create a Cloud Deploy release and rollout to deploy a Vertex AI model to the target.

## 1. Clone Repository

Clone this repository and navigate to the quickstart directory (`cloud-deploy-samples/custom-targets/vertex-ai/quickstart`) since the commands provided expect to be executed from that directory.

## 2. Environment variables

To simplify the commands in this quickstart, set the following environment variables with your values:

```shell
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="YOUR_REGION"
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


## 4. Import a model into Model Registry

We will upload a pre-existing model to Model Registry before deploying it with Cloud Deploy.
The `gcloud ai models upload` command requires a cloud storage path to the trained model, and a
docker image to run the model in.
For this quickstart, we will be using a pre-trained model from Vertex AI's cloud samples bucket.
For the docker image, we use a Cloud AI optimized version of tensorflow: [tf_opt](https://cloud.google.com/vertex-ai/docs/predictions/optimized-tensorflow-runtime).

   ```shell
   gcloud ai models upload \
       --artifact-uri gs://cloud-samples-data/vertex-ai/model-deployment/models/boston/model \
       --display-name=test-model \
       --container-image-uri=us-docker.pkg.dev/vertex-ai-restricted/prediction/tf_opt-cpu.nightly:latest \
       --project=$PROJECT_ID \
       --region=$REGION \
       --model-id=test_model
   ```

If this is your first time using Vertex AI in this project, this operation will
take 10 minutes or so.

## 5. Create a Vertex AI Endpoint

Create a Vertex AI endpoint using the following commands:

   ```shell
   export ENDPOINT_ID="prod"
   gcloud ai endpoints create --display-name prod-endpoint --endpoint-id $ENDPOINT_ID --region $REGION --project $PROJECT_ID
   ```

If this is your first time using Vertex AI in this project, this operation will
take 5 minutes or so.

The endpoint ID will be used to refer to the endpoint, rather than the display name.

## 6. Build and Register a Custom Target Type for Vertex AI

From within the `quickstart` directory, run this command to build the Vertex AI model deployer image and
install the custom target resources:

```shell
../build_and_register.sh -p $PROJECT_ID -r $REGION
```

For information about the `build_and_register.sh` script, see the [README](../README.md#build)

## 7. Create delivery pipeline, target, and skaffold

Similarly, within the `quickstart` directory, run this second command to replace placeholders in `clouddeploy.yaml`
and `configuration/skaffold.yaml` with actual values

```shell
./replace_variables.sh -p $PROJECT_ID -r $REGION -e $ENDPOINT_ID
```

The command does the following:
1. Replaces the placeholders in `clouddeploy.yaml`
2. Obtains the URL of the latest version of the custom image, built in step 6, and sets it in `configuration/skaffold.yaml`


Lastly, apply the Cloud Deploy configuration defined in `clouddeploy.yaml`:

```shell
gcloud deploy apply --file=clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

## 8. Create a release and rollout

Create a Cloud Deploy release for the configuration defined in the `configuration` directory. This automatically
creates a rollout that deploys the first model version to the target.

```shell
gcloud deploy releases create release-001 \
    --delivery-pipeline=vertex-ai-cloud-deploy-pipeline \
    --project=$PROJECT_ID \
    --region=$REGION \
    --source=configuration \
    --deploy-parameters="customTarget/vertexAIModel=projects/$PROJECT_ID/locations/$REGION/models/test_model"
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

## 9. Monitor rollout status

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
## 10. Inspect aliases in the deployed model 

Monitor the post-deploy operation by querying the rollout:

```
gcloud deploy rollouts describe release-001-to-prod-endpoint-0001 --release=release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION --format "(phases[0].deploymentJobs.postdeployJob)"
```

After the post-deploy job has succeeded, you can then inspect the deployed model and view its currently assigned aliases. `prod` and `champion` should be assigned.

```shell
gcloud ai models describe test_model --region $REGION --project $PROJECT_ID --format "(versionAliases)"
```

## 11. Clean up

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
gcloud deploy delete --file=clouddeploy.yaml --force --project=$PROJECT_ID --region=$REGION
```
