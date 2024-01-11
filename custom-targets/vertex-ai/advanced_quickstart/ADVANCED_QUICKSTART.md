# Cloud Deploy Vertex AI Model Deployer Advanced Quickstart

## Overview

This quickstart is a more advanced version of this [Quickstart](../quickstart/QUICKSTART.md) which showcases additional cloud deploy features.

In this quickstart you will:

1. Import multiples Vertex AI models into model registry and create two endpoints: a development and production endpoint.
2. Define a Cloud Deploy delivery pipeline, custom target type for Vertex AI, and two targets.
3. Create an initial Cloud Deploy Release, deploy it to the dev target adn then promote it to the prod target.
4. Create a second Cloud Deploy release, deploy it to the dev target, and then use canary strategy to gradually roll it out to the production target.

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

4. Build and Register a Custom Target Type for Vertex AI

From within the `quickstart` directory, run this command to build the Vertex AI model deployer image and
install the custom target resources:

```shell
../build_and_register.sh -p $PROJECT_ID -r $REGION
```
For information about the `build_and_register.sh` script, see the [README](../README.md#build)


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

## 5. Create Vertex AI Endpoints

Create the Vertex AI endpoints using the following commands:

   ```shell
   export DEV_ENDPOINT_ID="dev"
   gcloud ai endpoints create --display-name $DEV_ENDPOINT_ID --endpoint-id $DEV_ENDPOINT_ID --region $REGION --project $PROJECT_ID
   ```

   ```shell
   export PROD_ENDPOINT_ID="prod"
   gcloud ai endpoints create --display-name $PROD_ENDPOINT_ID --endpoint-id $PROD_ENDPOINT_ID --region $REGION --project $PROJECT_ID
   ```

If this is your first time using Vertex AI in this project, this operation will
take 5 minutes or so.

The endpoint ID will be used to refer to the endpoint, rather than the display name.

## 6. Create delivery pipeline, targets, and skaffold

Within the `quickstart` directory, run this command to make a temporary copy of `clouddeploy.yaml` and
`configuration/skaffold.yaml`, and to replace placeholders in the copies with actual values

```shell
export TMPDIR=$(mktemp -d)
./replace_variables.sh -p $PROJECT_ID -r $REGION -e $ENDPOINT_ID -t $TMPDIR
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
creates a rollout that deploys the first model version to the development target.

```shell
gcloud deploy releases create release-001 \
    --delivery-pipeline=vertex-ai-cloud-deploy-pipeline \
    --project=$PROJECT_ID \
    --region=$REGION \
    --source=$TMPDIR/configuration \
    --deploy-parameters="customTarget/vertexAIModel=projects/$PROJECT_ID/locations/$REGION/models/test_model@1"
```

### Explanation of command line flags

The `--source` command line flag instructs gcloud where to look for the configuration files relative to the working directory where the command is run.

The `--deploy-parameters` flag is used to provide the custom deployer with additional parameters needed to perform the deployment.

Here, we are providing the custom deployer with deploy parameter `customTarget/vertexAIModel`
which specifies the full resource name of the model to deploy. We are also specifying
the model version by including it at the end of the model name.

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
gcloud deploy rollouts describe release-001-to-dev-endpoint-0001 --release=release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION
```

It will take up to 15 minutes for the model to fully deploy.

After the rollout completes, you can inspect the deployed models and traffic splits of the endpoint with `gcloud`

```shell
gcloud ai endpoints describe $DEV_ENDPOINT_ID --region $REGION --project $PROJECT_ID
```
## 9. Inspect aliases in the deployed model

Monitor the post-deploy operation by querying the rollout:

```
gcloud deploy rollouts describe release-001-to-dev-endpoint-0001 --release=release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION --format "(phases[0].deploymentJobs.postdeployJob)"
```

After the post-deploy job has succeeded, you can then inspect the deployed model and view its currently assigned aliases. `dev` and `challenger` should be assigned.

```shell
gcloud ai models describe test_model@1 --region $REGION --project $PROJECT_ID --format "(versionAliases)"
```

## 10. Deploy to production

Promote the model in `dev` endpoint to `prod` by using the `gcloud deploy releases promote command`

Since this is the first deployment to that target, we will skipping directly to the `stable` phase, where 100% of the traffic
goes to the model.
```shell
gcloud deploy releases promote --release release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION --starting-phase-id stable
```

Monitor the rollout progress using this command:

```
gcloud deploy rollouts describe release-001-to-prod-endpoint-0001 --release=release-001 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION --format "(state)"
```

After the rollout completes, verify that the model has been deployed to the production endpoint:

```shell
gcloud ai endpoints describe $PROD_ENDPOINT_ID --region $REGION --project $PROJECT_ID
```

After the post-deploy job has succeeded, inspect the deployed model and view its currently assigned aliases. `prod` and `champion` should be assigned.

```shell
gcloud ai models describe test_model@1 --region $REGION --project $PROJECT_ID --format "(versionAliases)"
```

## 11. Upload a new model version and create new release

Create a new version of the model by re-uploading the boston dataset with `gcloud`. 
This time however, use the `parent-model` flag so that the model is uploaded as
a new version of the first model rather than a separate model.
   ```shell
   gcloud ai models upload \
       --artifact-uri gs://cloud-samples-data/vertex-ai/model-deployment/models/boston/model \
       --display-name=test-model \
       --container-image-uri=us-docker.pkg.dev/vertex-ai-restricted/prediction/tf_opt-cpu.nightly:latest \
       --project=$PROJECT_ID \
       --region=$REGION \
       --parent-model=projects/$PROJECT_ID/locations/$REGION/models/test_model
   ```

After the upload completes, create a new cloud deploy release using the new model
version. This time, changing the version ID of the vertexAIModel deploy parameter
to 2 to refer to the new model.

```shell
gcloud deploy releases create release-002 \
    --delivery-pipeline=vertex-ai-cloud-deploy-pipeline \
    --project=$PROJECT_ID \
    --region=$REGION \
    --source=configuration \
    --deploy-parameters="customTarget/vertexAIModel=projects/$PROJECT_ID/locations/$REGION/models/test_model@2"
```

Inspect the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-002-to-dev-endpoint-0001 --release=release-002 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION
```

After the post-deploy job has succeeded, inspect the deployed model and view its currently assigned aliases. `dev` and `challenger` should be assigned.

```shell
gcloud ai models describe test_model@2 --region $REGION --project $PROJECT_ID --format "(versionAliases)"
```

## 12. Promote the second release

Promote the second release to the production endpoint, since this is the second release,
deployed to the target, only 50% of the traffic will be initially routed to it.

```shell
gcloud deploy releases promote --release release-002 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION
```

Monitor the rollout progress with the following command:
```shell
gcloud deploy rollouts describe release-002-to-prod-endpoint-0001 --release=release-002 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION
```
After the rollout completes the first phase, inspect the endpoint traffic. It should
have a 50/50 traffic split between both model versions:

```shell
gcloud ai endpoints describe $PROD_ENDPOINT_ID --region $REGION --project $PROJECT_ID
```

Finally, advance the rollout to the final `stable` phase:

```shell
gcloud deploy rollouts advance release-002-to-prod-endpoint-0001 --release=release-002 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION
```

Monitor the rollout progress with the following command:
```shell
gcloud deploy rollouts describe release-002-to-prod-endpoint-0001 --release=release-002 --delivery-pipeline=vertex-ai-cloud-deploy-pipeline --project=$PROJECT_ID --region=$REGION --format "(phases[1])"
```

After the rollout completes, inspect the production endpoint to verify that 100%
of the traffic has been routed to the new model version:
```shell
gcloud ai endpoints describe $PROD_ENDPOINT_ID --project $PROJECT_ID --region $REGION 
```

After the post-deploy job has succeeded, inspect the deployed model and view its currently assigned aliases. `prod` and `champion` should be assigned.

```shell
gcloud ai models describe test_model@2 --region $REGION --project $PROJECT_ID --format "(versionAliases)"
```

## 13. Clean up

To delete the endpoints after the quickstart, run the following commands:

Obtain the id of the deployed model in the development endpoint:
```shell
gcloud ai endpoints describe $DEV_ENDPOINT_ID --region $REGION --project $PROJECT_ID --format "(deployedModels[0].id)"
```

Undeploy the model:
```shell
gcloud ai endpoints undeploy-model $DEV_ENDPOINT_ID --region $REGION --project $PROJECT_ID --deployed-model-id $DEPLOYED_MODEL_ID
```

Delete the endpoint:
```shell
gcloud ai endpoints delete $DEV_ENDPOINT_ID --region $REGION --project $PROJECT_ID
```

Obtain the id of the deployed model in the production endpoint:
```shell
gcloud ai endpoints describe $PROD_ENDPOINT_ID --region $REGION --project $PROJECT_ID --format "(deployedModels[0].id)"
```

Undeploy the model:
```shell
gcloud ai endpoints undeploy-model $PROD_ENDPOINT_ID --region $REGION --project $PROJECT_ID --deployed-model-id $DEPLOYED_MODEL_ID
```

Delete the endpoint:
```shell
gcloud ai endpoints delete $PROD_ENDPOINT_ID --region $REGION --project $PROJECT_ID
```

To delete the imported model:

```shell
gcloud ai models delete test_model --region $REGION --project $PROJECT_ID
```

To delete Cloud Deploy resources:

```shell
gcloud deploy delete --file=clouddeploy.yaml --force --project=$PROJECT_ID --region=$REGION
```
