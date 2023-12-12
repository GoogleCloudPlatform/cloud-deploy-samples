# Cloud Deploy Infrastructure Manager Deployer Quickstart

This page shows you how to use Cloud Deploy to deploy Google Cloud infrastructure resources using the sample
Infrastructure manager deployer. This quickstart uses Terraform configuration that will create two Google Compute
Networks, one for a dev and prod environment both in the same project.

In this quickstart you will:

1. Define a Cloud Deploy delivery pipeline, custom target type for Infrastructure Manager, and two targets (dev and prod).
2. Create a Cloud Deploy release and rollout to deploy the Terraform configuration via Infrastructure Manager to the dev target.
3. Promote the release to deploy the Terraform configuration via Infrastructure Manager to the prod target.

## 1. Clone Repository

Clone this repository and navigate to the quickstart directory (`cloud-deploy-samples/custom-targets/infrastructure-manager/quickstart`) since the commands provided expect to be executed from that directory.

## 2. Environment Variables

To simplify the commands in this quickstart, set the following environment variables with your values:

```shell
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="YOUR_REGION"
```

> [!NOTE]
> This quickstart uses the same region for Cloud Deploy and Infrastructure Manager for simplicity reasons. Since Infrastructure Manager is in a subset of Cloud Deploy regions please ensure to pick a [supported Infrastructure Manager region](https://cloud.google.com/infrastructure-manager/docs/locations).

## 3. Prerequisites

### APIs
Enable the Cloud Deploy API, Infrastructure Manager API, and Compute Engine API.

```shell
gcloud services enable clouddeploy.googleapis.com config.googleapis.com compute.googleapis.com --project $PROJECT_ID
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

2. `iam.serviceAccountUser` role, to give the service account `actAs` permission:

    ```shell
    gcloud iam service-accounts add-iam-policy-binding $(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/iam.serviceAccountUser" \
    --project=$PROJECT_ID
    ```

3. `config.agent` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/config.agent"
    ```

4. `config.admin` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/config.admin"
    ```

5. `compute.networkAdmin` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/compute.networkAdmin"
    ```

## 4. Build and Register a Custom Target Type for Infrastructure Manager
From within the `quickstart` directory, run the following command to build the Infrastructure Manager deployer image and register a Cloud Deploy custom target type that references the image:

```shell
../build_and_register.sh -p $PROJECT_ID -r $REGION
```

For information about the `build_and_register.sh` script, see the [README](../README.md#build)

## 5. Create delivery pipeline and targets
Replace the placeholders in the `clouddeploy.yaml` in this directory with your environment variable values:

```shell
sed -i "s/\$PROJECT_ID/${PROJECT_ID}/g" ./clouddeploy.yaml
sed -i "s/\$REGION/${REGION}/g" ./clouddeploy.yaml
```

Apply the Cloud Deploy configuration:

```shell
gcloud deploy apply --file=clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

## 6. Create a release
Create a Cloud Deploy release for the configuration defined in the `configuration` directory. This will automatically create a rollout that deploys the Terraform configuration to the dev target.

```shell
gcloud deploy releases create release-001 --delivery-pipeline=im-network-pipeline --source=configuration --project=$PROJECT_ID --region=$REGION
```

### Configuration context
The Terraform configuration is structured so the dev and prod root modules are defined in their own directories:

* `configuration/dev`
* `configuration/prod`

The path to the relevant configuration is provided as a deploy parameter on the delivery pipeline stage. Additionaly, the configuration expects `project_id` variable to be set, this is provided as a deploy parameter on the targets.

## 7. Check rollout status for dev target
In the Cloud Deploy UI for your project click on the `im-network-pipeline` delivery pipeline. Here you can see the release created and the rollout to the dev target for the release.

You can also describe the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-001-to-im-dev-0001 --release=release-001 --delivery-pipeline=im-network-pipeline --project=$PROJECT_ID --region=$REGION
```

Once the rollout has succeeded the dev Terraform configuration has been applied. This will have created an Infrastructure Manager Deployment, which can be described with the following command:

```shell
gcloud infra-manager deployments describe dev-vpc-network --project=$PROJECT_ID --location=$REGION
```

The network resource created by Infrastructure Manager for the Terraform configuration can be described with the following command:

```shell
gcloud compute networks describe im-ct-quickstart-dev-network --project=$PROJECT_ID
```

## 8. Promote the Release
Promote the release to start a rollout for the prod target.

```shell
gcloud deploy releases promote --release=release-001 --delivery-pipeline=im-network-pipeline --project=$PROJECT_ID --region=$REGION
```

## 9. Check rollout status for prod target
View the `im-network-pipeline` delivery pipeline in the Cloud Deploy UI.

To describe the prod rollout run the following command:

```shell
gcloud deploy rollouts describe release-001-to-im-prod-0001 --release=release-001 --delivery-pipeline=im-network-pipeline --project=$PROJECT_ID --region=$REGION
```

Once the rollout has succeeded the prod Terraform configuration has been applied. This will have created an Infrastructure Manager Deployment, which can be described with the following command:

```shell
gcloud infra-manager deployments describe prod-vpc-network --project=$PROJECT_ID --location=$REGION
```

The network resource created by Infrastructure Manager for the Terraform configuration can be described with the following command:

```shell
gcloud compute networks describe im-ct-quickstart-prod-network --project=$PROJECT_ID
```

## 10. Clean up

Delete the Infrastructure Manager Deployments: `dev-vpc-network` and `prod-vpc-network`. Infrastructure Manager will delete the resources it manages, i.e. the compute networks created:

```shell
gcloud infra-manager deployments delete dev-vpc-network --project=$PROJECT_ID --location=$REGION
```

```shell
gcloud infra-manager deployments delete prod-vpc-network --project=$PROJECT_ID --location=$REGION
```

Delete the Cloud Deploy resources:
```
gcloud deploy delete --file=clouddeploy.yaml --force --project=$PROJECT_ID --region=$REGION
```
