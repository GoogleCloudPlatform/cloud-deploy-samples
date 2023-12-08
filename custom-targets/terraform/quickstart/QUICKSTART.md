# Cloud Deploy Terraform Deployer Quickstart

This page shows you how to use Cloud Deploy to deploy Google Cloud infrastructure resources using the sample
Terraform deployer. This quickstart uses Terraform configuration that will create two Google Compute Networks, 
one for a dev and prod environment both in the same project.

In this quickstart you will:

1. Create two Cloud Storage buckets to store the Terraform state for the dev and prod resources.
2. Define a Cloud Deploy delivery pipeline, custom target type for Terraform, and two targets (dev and prod).
3. Create a Cloud Deploy release and rollout to deploy the Terraform configuration to the dev Terraform custom target.
4. Promote the release to deploy the Terraform configuration to the prod Terraform custom target.

## 1. Clone Repository

Clone this repository and navigate to the quickstart directory (`cloud-deploy-samples/custom-targets/terraform/quickstart`) since the commands provided expect to be executed from that directory.

## 2. Environment Variables

To simplify the commands in this quickstart, set the following environment variables with your values:

```shell
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="YOUR_REGION"
```

## 3. Prerequisites

### APIs
Enable the Cloud Deploy API and Compute Engine API.

```shell
gcloud services enable clouddeploy.googleapis.com compute.googleapis.com --project $PROJECT_ID
```

### Permissions
Make sure the default compute service account, `{project_number}-compute@developer.gserviceaccount.com`, used by Cloud Deploy has sufficient permissions:

1. `clouddeploy.jobRunner` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"
    ```

2. `storage.objectUser` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/storage.objectUser"
    ```

3. `compute.networkAdmin` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/compute.networkAdmin"
    ``` 

## 4. Build and Register a Custom Target Type for Terraform
From within the `quickstart` directory, run the following command to build the Terraform deployer image and register a Cloud Deploy custom target type that references the image:

```shell
../build_and_register.sh -p $PROJECT_ID -r $REGION
```

For information about the `build_and_register.sh` script, see the [README](../README.md#build)

## 5. Create Cloud Storage buckets for Terraform backends

You will need to create two Cloud Storage buckets that will be configured on the Cloud Deploy dev and prod Terraform custom targets. Use the following commands to create the buckets and set environment variables for each one:

```shell
gcloud storage buckets create gs://$PROJECT_ID-$REGION-tf-dev-backend --project $PROJECT_ID --location $REGION
export DEV_BACKEND_BUCKET=$PROJECT_ID-$REGION-tf-dev-backend
```

```shell
gcloud storage buckets create gs://$PROJECT_ID-$REGION-tf-prod-backend --project $PROJECT_ID --location $REGION
export PROD_BACKEND_BUCKET=$PROJECT_ID-$REGION-tf-prod-backend
```

## 6. Create delivery pipeline and targets
Replace the placeholders in the `clouddeploy.yaml` in this directory with your environment variable values:

```shell
sed -i "s/\$PROJECT_ID/${PROJECT_ID}/g" ./clouddeploy.yaml
sed -i "s/\$REGION/${REGION}/g" ./clouddeploy.yaml
sed -i "s/\$DEV_BACKEND_BUCKET/${DEV_BACKEND_BUCKET}/g" ./clouddeploy.yaml
sed -i "s/\$PROD_BACKEND_BUCKET/${PROD_BACKEND_BUCKET}/g" ./clouddeploy.yaml
```

Apply the Cloud Deploy configuration:

```shell
gcloud deploy apply --file=clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

## 7. Create a release
Create a Cloud Deploy release for the configuration defined in the `configuration` directory. This will automatically
create a rollout that deploys the Terraform configuration to the dev target.

```shell
gcloud deploy releases create release-001 --delivery-pipeline=tf-network-pipeline --project=$PROJECT_ID --region=$REGION --source=configuration --deploy-parameters="customTarget/tfEnableRenderPlan=true"
```

### Configuration context
The Terraform configuration is structured so the dev and prod root modules are defined in their own directories: 

* `configuration/environments/dev`
* `configuration/environments/prod`

Both environment root modules have a child module defined that is sourced from `configuration/network-module`. Additionally, the
configuration expects `project_id` variable to be set, this is provided as a deploy parameter on the targets.

## 8. Check rollout status for dev target
In the [Cloud Deploy UI](https://console.cloud.google.com/deploy/delivery-pipelines) for your project click on the `tf-network-pipeline` delivery pipeline. Here you can see the release created and the rollout to the dev target for the release.

You can also describe the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-001-to-tf-dev-0001 --release=release-001 --delivery-pipeline=tf-network-pipeline --project=$PROJECT_ID --region=$REGION
```

Once the rollout has succeeded the dev Terraform configuration has been applied. This will have created a compute network
resource named `tf-ct-quickstart-dev-network`. To describe the resource, run:

```shell
gcloud compute networks describe tf-ct-quickstart-dev-network --project=$PROJECT_ID
```

## 9. Promote the release
Promote the release to start a rollout for the prod Terraform custom target:

```shell
gcloud deploy releases promote --release=release-001 --delivery-pipeline=tf-network-pipeline --project=$PROJECT_ID --region=$REGION
```

## 10. Check rollout status for prod target
View the `tf-network-pipeline` delivery pipeline in the [Cloud Deploy UI](https://console.cloud.google.com/deploy/delivery-pipelines).

To describe the prod rollout run the following command:

```shell
gcloud deploy rollouts describe release-001-to-tf-prod-0001 --release=release-001 --delivery-pipeline=tf-network-pipeline --project=$PROJECT_ID --region=$REGION
```

Once the rollout has succeeded the prod Terraform configuration has been applied. This will have created a compute network
resource named `tf-ct-quickstart-prod-network`. To describe the resource, run:

```shell
gcloud compute networks describe tf-ct-quickstart-prod-network --project=$PROJECT_ID
```

## 11. Clean up

Delete the Cloud Storage objects and buckets:

```shell
gcloud storage rm -r gs://$PROJECT_ID-$REGION-tf-dev-backend gs://$PROJECT_ID-$REGION-tf-prod-backend
```

Delete the compute networks:

```shell
gcloud compute networks delete tf-ct-quickstart-dev-network tf-ct-quickstart-prod-network --project=$PROJECT_ID
```

Delete the Cloud Deploy resources:

```shell
gcloud deploy delete --file=clouddeploy.yaml --force --project=$PROJECT_ID --region=$REGION
```
