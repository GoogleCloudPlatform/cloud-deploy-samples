# Cloud Deploy Datadog Analysis Sample

This directory contains a sample implementation of a Cloud Deploy analysis
custom check for Datadog.

If this container reports a failure, the rollout is marked as
[FAILED](https://cloud.google.com/deploy/docs/deployment-strategies/manage-rollout#rollout_states).
Otherwise, the container is run again depending on how often and for how long
you've configured the analysis check.

## Overview

The container is go module that takes in environment variables, defined in your
Cloud Deploy delivery pipeline configuration file, and uses them to call the
Datadog API, specifically the
[SearchEvents API](https://docs.datadoghq.com/api/latest/events/#search-events).
The container uses the response from the Datadog API to determine if there are
alerts firing, and if so will report back a failure. For example, if you've set
up a few monitors on Datadog, you'd provide them as the `Query` environment
variable, and the container queries Datadog to determine if any of the monitors
have started firing since the rollout began.

The query timeframe looks at any alerts that are firing from the time the
rollout began to the time of the check. It only reports back a failure if the
alert is **still firing**.

For example, say a rollout started at 2pm. An alert began firing at 2:15pm and
is still firing. The analysis check is done at 2:30pm. That check sees the
firing alert and reports back a failure. If on the other hand the alert only
fired from 2:15-2:20pm and is now in an OK state, it would not report a failure.

For more information on the analysis feature and using custom analysis checks, 
look at the [documentation](https://docs.cloud.google.com/deploy/docs/analysis).

If you want to use deploy parameters to use with environment variables such as 
image, look at this [documentation](https://docs.cloud.google.com/deploy/docs/parameters).


## Assumptions

This README assumes:

1.  You've already
    [set up Datadog](https://docs.datadoghq.com/integrations/google-cloud-platform/?tab=organdfolderlevelprojectdiscovery)
    to monitor your GCP resources.

2.  You have an existing Cloud Deploy delivery pipeline setup, and you'd like to
    add a final analysis stage to that job. If you don't have a pipeline, see
    our [GKE](https://cloud.google.com/deploy/docs/deploy-app-gke) or
    [Cloud Run](https://cloud.google.com/deploy/docs/deploy-app-run) quickstart.

## Environment Variables

To simplify the commands in this README, set the following environment variables
with your values:

```shell
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="YOUR_REGION"
export PIPELINE_NAME="YOUR_PIPELINE_NAME"
```

## Permissions

Make sure the default compute service account,
`{project_number}-compute@developer.gserviceaccoumt.com`, used by Cloud Deploy
has sufficient permissions:

1.  `clouddeploy.jobRunner` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"
    ```

2.  `secretmanager.secretAccessor` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/secretmanager.secretAccessor"
    ```

3.  GCS storage permissions - this sample uploads metadata to a GCS bucket:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/storage.objectUser"
    ```

## Setup Secret Manager {#secrets}

In order to authenticate to the Datadog API, an
[API key and application key](https://docs.datadoghq.com/account_management/api-app-keys/)
are required. These should be in the "Personal Settings" section of Datadog. The
application key **must** have the `events_read` permission.

For this sample to work, the API key and application key need to be saved in
Google Cloud's Secret Manager. The container retrieves them to use them for
authentication.

[Here](https://cloud.google.com/secret-manager/docs/creating-and-accessing-secrets)
are the instructions on how to create a secret. Once you've created two secrets,
one with the API key and one with the application key, get the names of the
policy. Within a version click on `Actions > Copy Resource Name`. It will look
like `projects/123456/secrets/datadog-apikey/versions/1`.

Update the environment variables `DATADOG_APIKEY=` and `DATADOG_APPKEY=` in the
analysis stanza of your clouddeploy.yaml. See the clouddeploy.yaml file in the
configuration directory as an example.

## Build the image

First, create an Artifact Registry repository to store the image and set up
Docker authentication for that repository. In the following command, tthe repo
is named `datadog-container`, but feel free to name it whatever you want.

```shell
gcloud artifacts repositories create datadog-container \
    --location "$REGION" --project "$PROJECT_ID" \
    --repository-format docker
gcloud -q auth configure-docker $REGION-docker.pkg.dev
```

Next, give the default Compute service account access to read from this
repository:

```shell
gcloud -q artifacts repositories add-iam-policy-binding \
    --project "${PROJECT_ID}" --location "${REGION}" datadog-container \
    --member=serviceAccount:$(gcloud -q projects describe $PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.reader"
```

Finally, set the image's location in an environment variable for use in future
steps, then build and push the image. Run the following command from the
top-level `cloud-deploy-samples` directory.

```shell
IMAGE=$REGION-docker.pkg.dev/$PROJECT_ID/datadog-container/analysis
docker build . -f "./analysis/datadog/Dockerfile" --tag $IMAGE
docker push $IMAGE
```

## Update environment variables in your Cloud Deploy config

There are four required environment variables that need to be set in your Cloud
Deploy configuration file in the analysis stanza:

Variable           | Required | Description
------------------ | -------- | -------------------------------------------
`DatadogAPISecret` | Yes      | Datadog API Secret stored in Secret Manager
`DatadogAppSecret` | Yes      | Datadog App Secret stored in Secret Manager
`Query`            | Yes      | Quer(ies) to use to search for alerts
`DatadogURL`       | Yes      | Datadog site you use

Please refer to the [Secret Manager section](#secrets) above on the
`DatadogAPISecret` and `DatadogAppSecret`.

**Query**

*   At least *one query* must be provided.
*   If you have multiple queries, you must suffix them differently. For example
    `Query_foo`, `Query_bar`, etc. We include the query in the logs, and that
    way you can easily see which query was the one that failed. The suffix can
    be anything as long as it starts with `Query`.
*   There will be one API call per query.
*   You can't control the order in which queries are executed. It's assumed that
    the queries are independent and there are no dependencies among them.

The queries can actually be a little tricky to get right so there are several
examples in this section. Specifically, for a `monitor_id` you must include the
`@` symbol before `monitor_id`. In order to get your monitor ID, click on your
monitor in Datadog, and the number is in the URL. For example, if the URL is
https://us5.datadoghq.com/monitors/123456, the ID is 123456.
[Here](https://docs.datadoghq.com/service_management/events/explorer/searching/)
is documentation from Datadog on query syntax.

**Query examples:**

| Type                            | Query                                    |
| :------------------------------ | :--------------------------------------- |
| Specific monitor                | "@monitor_id:11825239"                   |
| Multiple monitors               | "@monitor_id:12345 OR                    |
:                                 : @monitor_id\:789101"                     :
| Labels                          | "foo:bar"                                |
| Location                        | "location:us-west2"                      |
| GKE cluster                     | "cluster:canary-quickstart-cluster"      |
| GKE deployment                  | "kube_deployment:my-deployment-canary"   |
| Cloud Run revision              | "revision_name:datadog-service-dev-1234" |
| Cloud Run revision and          | "revision_name:datadog-service-dev-1234  |
: monitor_id_                     : AND @monitor_id\:11825239"               :
| Cloud Run revision using system | "revision_name:${{                       |
: parameter                       : render.metadata.cloud_run.revision.name  :
:                                 : }}"                                      :

Note that `status:error` is appended to every query so that the analysis looks
at only alerts that are firing. The last of these query examples uses a Cloud
Run system parameter, but you can use any system parameter.

**DatadogURL**

Use one of the
[official URLs](https://docs.datadoghq.com/getting_started/site/#access-the-datadog-site)
from Datadog. For example, "https://us5.datadoghq.com". This is used to
construct the URL to the alert and include it in the logs so you can easily see
what alert was firing.

## Update your Cloud Deploy config and create a release

1.  Take a look at the `clouddeploy.yaml` example in the configuration
    directory, and update your Cloud Deploy config with a new analysis stanza
    and your values. You can configure how long the analysis job will run and
    how often a check is performed.

2.  Apply your changes:

    ```shell
    gcloud deploy apply \
      --file clouddeploy.yaml \
      --project=${PROJECT_ID} \
      --region=${REGION}
    ```

3.  Create a release. Note that analysis runs at the end, so it may take a few
    minutes until you can see the analysis job in progress.

```shell
gcloud deploy releases create release-01 \
  --project ${PROJECT_ID} \
  --region ${REGION} \
  --delivery-pipeline ${PIPELINE_NAME}
```