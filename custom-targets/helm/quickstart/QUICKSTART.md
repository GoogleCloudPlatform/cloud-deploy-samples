# Cloud Deploy Helm Deployer Quickstart

This page shows you how to use Cloud Deploy to deploy to a Google Kubernetes Engine (GKE) cluster using the sample Helm deployer. This quickstart uses a Helm chart that contains a Kubernetes `Deployment` with a `nginx` image.

In this quickstart you will:

1. Create a GKE cluster.
2. Define a Cloud Deploy delivery pipeline, custom target type for Helm, and a target.
3. Create a Cloud Deploy release and rollout to deploy the Helm chart to the Helm custom target.

## 1. Clone Repository

Clone this repository and navigate to the quickstart directory (`cloud-deploy-samples/custom-targets/helm/quickstart`) since the commands provided expect to be executed from that directory.

## 2. Environment variables

To simplify the commands in this quickstart, set the following environment variables with your values:

```shell
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="YOUR_REGION"
```

## 3. Prerequisites

### APIs
Enable the Cloud Deploy API and Kubernetes Engine API.

```shell
gcloud services enable clouddeploy.googleapis.com container.googleapis.com --project $PROJECT_ID
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

2. `container.developer` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/container.developer"
    ```

## 4. Build and Register a Custom Target Type for Helm
From within the `quickstart` directory, run the following command to build the Helm deployer image and register a Cloud Deploy custom target type that references the image:

```shell
../build_and_register.sh -p $PROJECT_ID -r $REGION
```

For information about the `build_and_register.sh` script, see the [README](../README.md#build)

## 5. Create a Google Kubernetes Engine cluster
Create a GKE cluster for the Helm custom target. Use the following command to create the cluster and set the ID as an environment variable for a future step:

```shell
export CLUSTER_ID=quickstart-cluster-helm
gcloud container clusters create-auto $CLUSTER_ID --project=$PROJECT_ID --region=$REGION
```

## 6. Create delivery pipeline and target
Replace the placeholder in the `clouddeploy.yaml` in this directory with your environment variable value:

```shell
sed -i "s/\$PROJECT_ID/${PROJECT_ID}/g" ./clouddeploy.yaml
sed -i "s/\$REGION/${REGION}/g" ./clouddeploy.yaml
sed -i "s/\$CLUSTER_ID/${CLUSTER_ID}/g" ./clouddeploy.yaml
```

Apply the Cloud Deploy configuration:

```shell
gcloud deploy apply --file=clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

## 7. Create a release
Create a Cloud Deploy release for the configuration defined in the `configuration` directory. This will automatically create a rollout that deploys the Helm chart to the target.

```shell
gcloud deploy releases create release-001 --delivery-pipeline=helm-pipeline --project=$PROJECT_ID --region=$REGION --source=configuration
```

### Configuration context
The Helm chart is in `configuration/mychart`. The path to the Helm chart within the `configuration` directory was configured as a deploy parameter on the pipeline stage, see the `clouddeploy.yaml` file.

## 8. Check rollout status for the target
In the [Cloud Deploy UI](https://console.cloud.google.com/deploy/delivery-pipelines) for your project click on the `helm-pipeline` delivery pipeline. Here you can see the release created and the rollout to the target for the release.

You can also describe the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-001-to-helm-cluster-0001 --release=release-001 --delivery-pipeline=helm-pipeline --project=$PROJECT_ID --region=$REGION
```

Once the rollout has succeeded the Kubernetes `Deployment` in the Helm chart has been deployed to the GKE cluster. See the [GKE Workload UI](https://console.cloud.google.com/kubernetes/workload/overview) for your project.

## 9. Clean up

Delete the GKE cluster:

```shell
gcloud container clusters delete quickstart-cluster-helm --project=$PROJECT_ID --region=$REGION
```

Delete the Cloud Deploy resources:

```shell
gcloud deploy delete --file=clouddeploy.yaml --force --project=$PROJECT_ID --region=$REGION
```
