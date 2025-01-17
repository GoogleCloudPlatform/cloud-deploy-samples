# Kubernetes Resource Clean Up Quickstart

This contains source code for a container that can be used to clean up
Kubernetes resources that were deployed by Cloud Deploy. It should be used as a
[postdeploy hook](https://cloud.google.com/deploy/docs/hooks). A configuration
example is provided as part of this quickstart.

See [the README](../README.md) for more information on how the container works
and what configuration is available.

## 1. Clone Repository

Clone this repository and navigate to the quickstart directory
(`cloud-deploy-samples/postdeploy-hooks/k8s-cleanup/quickstart`) since the
commands provided expect to be executed from that directory.

## 2. Environment variables

To simplify the commands in this quickstart, set the following environment
variables with your values:

```shell
PROJECT_ID="YOUR_PROJECT_ID"
REGION="YOUR_REGION"
```

## 3. Prerequisites

### APIs

Enable the Cloud Deploy API and Kubernetes Engine API.

```shell
gcloud services enable clouddeploy.googleapis.com container.googleapis.com --project $PROJECT_ID
```

You cannot use this container with
[the Organization Policy that disables Cloud Deploy's automatic labels](https://cloud.google.com/deploy/docs/labels-annotations#disabling_automatic_labels).
This container relies on those automatic labels to find the relevant Kubernetes
resources.

### Permissions

Make sure the default compute service account,
`{project_number}-compute@developer.gserviceaccount.com`, used by Cloud Deploy
has sufficient permissions:

1.  `clouddeploy.jobRunner` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
        --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
        --role="roles/clouddeploy.jobRunner"
    ```

2.  `container.developer` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
        --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
        --role="roles/container.developer"
    ```

## 4. Build the image

First, create an Artifact Registry repository to store the image and set up
Docker authentication for that repository.

```shell
gcloud artifacts repositories create cd-k8s-cleanup \
    --location "$REGION" --project "$PROJECT_ID" \
    --repository-format docker
gcloud -q auth configure-docker $REGION-docker.pkg.dev
```

Next, give the default compute service account access to read from this
repository:

```shell
gcloud -q artifacts repositories add-iam-policy-binding \
    --project "${PROJECT}" --location "${REGION}" cd-k8s-cleanup \
    --member=serviceAccount:$(gcloud -q projects describe $PROJECT --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.reader"
```

Finally, set the image's location in an environment variable for use in future
steps, then build and push the image:

```shell
IMAGE=$REGION-docker.pkg.dev/$PROJECT_ID/cd-k8s-cleanup/k8s-cleanup
docker build --tag $IMAGE ..
docker push $IMAGE
```

## 5. Create a Google Kubernetes Engine cluster

Create a GKE cluster for the quickstart. Use the following command to create the
cluster and set the ID as an environment variable for a future step:

```shell
CLUSTER_ID=quickstart-k8s-cleanup
gcloud container clusters create-auto $CLUSTER_ID --project=$PROJECT_ID --region=$REGION
```

This will take some time, up to ten minutes.

## 6. Create delivery pipeline and target

The `configuration` directory contains a set of files to create Cloud Deploy
Pipeline and use that pipeline to deploy a workload into the newly-created GKE
cluster.

The [`clouddeploy.yaml` file](configuration/clouddeploy.yaml) contains a
definition for a Cloud Deploy pipeline with a single GKE target. It specifies a
postdeploy action named `cleanup-action`.

Run these commands to modify the GKE Target resource in that file to point it to
the GKE cluster:

```shell
sed -i "s%\$PROJECT_ID%${PROJECT_ID}%g" ./configuration/clouddeploy.yaml
sed -i "s%\$REGION%${REGION}%g" ./configuration/clouddeploy.yaml
sed -i "s%\$CLUSTER_ID%${CLUSTER_ID}%g" ./configuration/clouddeploy.yaml
```

Apply the Cloud Deploy configuration:

```shell
gcloud deploy apply --file=configuration/clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

The defintion for the postdeploy action lives in the [`skaffold.yaml`] file. It
tells Skaffold which container to run and which arguments to provide to that
container.

Replace the `$K8s_CLEANUP_IMAGE` placeholder in that file with the image that we
just built:

```shell
sed -i "s%\$K8S_CLEANUP_IMAGE%${IMAGE}%g" ./configuration/skaffold.yaml
```

## 7. Create a release

Create a Cloud Deploy release for the configuration defined in the
`configuration` directory. The
[`kubernetes.yaml` file](configuration/kubernetes.yaml), which is referenced
from the [`skaffold.yaml` file](configuration/skaffold.yaml) contains a single
Deployment resource named `k8s-cleanup-deployment-orig`. Creating this release
will use Cloud Deploy to create this Deployment on the cluster.

```shell
gcloud deploy releases create release-001 \
    --delivery-pipeline=k8s-cleanup-qs \
    --project=$PROJECT_ID \
    --region=$REGION \
    --source=configuration \
    --images=my-app-image=gcr.io/google-containers/nginx@sha256:f49a843c290594dcf4d193535d1f4ba8af7d56cea2cf79d1e9554f077f1e7aaa
```

Notice the --images= flag, which you use to replace the placeholder
(my-app-image) in the manifest with the specific, SHA-qualified image. Google
recommends that you templatize your manifests this way, and that you use
SHA-qualified image names at release creation.

[Open the Cloud Deploy UI](https://console.cloud.google.com/deploy), click on
the `k8s-cleanup-qs` pipeline, then the `release-001` release, and on the
rollout for this release. You'll notice the rollout has a "Postdeploy" job.

Click on the Postdeploy job to inspect the logs. Towards the bottom of the logs,
you'll see this message:

`[clean-up-image] There are no resources to delete`

Since this is the first release, this is expected.

## 8. Create a release with a different Deployment resource

Next, let's rename the Deployment resource in the `kubernetes.yaml` file:

```shell
sed -i "s/k8s-cleanup-deployment-orig/k8s-cleanup-deployment-new/" ./configuration/kubernetes.yaml
```

Now we'll make another release:

```shell
gcloud deploy releases create release-002 \
    --delivery-pipeline=k8s-cleanup-qs \
    --project=$PROJECT_ID \
    --region=$REGION \
    --source=configuration \
    --images=my-app-image=gcr.io/google-containers/nginx@sha256:f49a843c290594dcf4d193535d1f4ba8af7d56cea2cf79d1e9554f077f1e7aaa
```

When you navigate to the rollout for this release and look at the logs for the
Postdeploy job, now you'll see that there are a number of resources to be
deleted:

`[clean-up-image] Beginning to delete resources, there are 3 resources to
delete`

It will then proceed to delete the Deployment, the ReplicaSet, and the Pod from
the previous release.

## 9. Cleanup

Delete the Cloud Deploy resources we created in steps 6–8:

```shell
gcloud deploy delete --file=configuration/clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

Delete the GKE cluster we created in step 5:

```shell
gcloud container clusters delete quickstart-k8s-cleanup \
    --region=$REGION --project=$PROJECT_ID
```

And delete the Artifact Registry repo we created in step 4:

```shell
gcloud artifacts repositories delete cd-k8s-cleanup \
    --location=$REGION --project=$PROJECT_ID
```
