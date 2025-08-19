# Verify through Monitoring

This contains a sample verify container that can be used with Cloud Deploy
during Deployment Verification. This app ensures that a certain response class
for requests do not exceed a percentage threshold for a given amount of time. It
uses [MQL](https://cloud.google.com/monitoring/mql) and builds up a query to
send to the monitoring API.

Within the `cloud-deploy` folder, there are sample YAMLs.

1.  `clouddeploy.yaml`: Defines a single [Cloud Run
    Target](https://cloud.google.com/deploy/docs/deploy-app-run). Defines a
    Delivery Pipeline that references that Cloud Run Target and specifies
    [Automated Canary
    Strategy](https://cloud.google.com/deploy/docs/deployment-strategies/canary).
    1.  `run.yaml`: Defines the service to be deployed to the Target.
1.  `skaffold.yaml`: Defines the deployer, associated manifests, and the
    container configuration for verification.

## 1. Environment variables

To simplify the commands in this readme, set the following environment variables
with your values:

```shell
PROJECT_ID="YOUR_PROJECT_ID"
REGION="YOUR_REGION"
```

## 2. Prerequisites

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

Grant the default execution service account actAs permission to deploy workloads
into Cloud Run:

 ```shell
gcloud iam service-accounts add-iam-policy-binding $(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/iam.serviceAccountUser" \
    --project=$PROJECT_ID
    ```

Add the Cloud Run developer permissions:

 ```shell
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/run.developer"
    ```

See [this page](https://cloud.google.com/deploy/docs/iam-roles-permissions) for more information about
permissions used in Cloud Deploy.

### Artifact Registry Repo

First, create an Artifact Registry repository to store the image and set up
Docker authentication for that repository.

```shell
gcloud artifacts repositories create verify \
    --location "$REGION" --project "$PROJECT_ID" \
    --repository-format docker
gcloud -q auth configure-docker $REGION-docker.pkg.dev
```

Next, give the default compute service account access to read from this
repository:

```shell
gcloud -q artifacts repositories add-iam-policy-binding \
    --project "${PROJECT_ID}" --location "${REGION}" verify \
    --member=serviceAccount:$(gcloud -q projects describe $PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/artifactregistry.reader"
```

### Building and pushing the image to a repo

Finally, set the image's location in an environment variable for use in future
steps, then build and push the image:

```shell
cd ../
IMAGE=$REGION-docker.pkg.dev/$PROJECT_ID/verify/cloud-metrics
docker build . -f "./verify-evaluate-cloud-metrics/Dockerfile" --tag $IMAGE
docker push $IMAGE
cd ./verify-evaluate-cloud-metrics
```

## 3. Create delivery pipeline and target

The `configuration` directory contains a set of files to create a Cloud Deploy
Pipeline and use that pipeline to deploy a workload to Cloud Run.

The [`clouddeploy.yaml` file](configuration/clouddeploy.yaml) contains a
definition for a Cloud Deploy pipeline with a single Cloud Run target.

Run these commands to replace the placeholders in that file:

```shell
sed -i "s%\$PROJECT_ID%${PROJECT_ID}%g" ./configuration/clouddeploy.yaml
sed -i "s%\$REGION%${REGION}%g" ./configuration/clouddeploy.yaml
```

Apply the Cloud Deploy configuration:

```shell
gcloud deploy apply --file=configuration/clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

The definition for the postdeploy action lives in the [`skaffold.yaml`] file. It
tells Skaffold which container to run and which arguments to provide to that
container.

Replace the `$IMAGE` placeholder in that file with the image that we just built:

```shell
sed -i "s%\$IMAGE%${IMAGE}%g" ./configuration/skaffold.yaml
```

You can configure the following inputs within the `skaffold.yaml`:

*   `project`: the project to look for the metrics. This defaults to the env
    variable: `CLOUD_DEPLOY_PROJECT`. More environment variables can be viewed
    [here](https://cloud.google.com/deploy/docs/verify-deployment#available_environment_variables).
    *   `table-name`: the monitoring
        [tablename](https://cloud.google.com/monitoring/mql/reference#fetch-tabop)
        to fetch from.
*   `metric-type`: The [metric
    type](https://cloud.google.com/monitoring/mql/reference#metric-tabop) to get
    from the table-name.
*   `predicates`: Commma delimited list of
    [predicates](https://cloud.google.com/monitoring/mql/reference#filter-tabop)
    to be applied in the query
*   `response-code-class`: The response_code_class to monitor for the error
    condition. Default is `5xx`.
*   `max-error-percentage`: The maximum allowable percentage of the specified
    response_code_class in a sliding window. Default is `10`.
*   `sliding-window`: The duration of the sliding window during the query.
    Default is `1m`.
*   `trigger-duration`: The duration required to observe the error condition for
    verify to fail. Default is `5m`.
*   `time-to-monitor`: The time to run this verification container for. If the
    time-to-monitor expires and there are no error conditions that has lasted >=
    the length of the trigger duration, this verification is marked as
    successful. Default is `20m`.
*   `refresh-period`: The time to wait before refreshing the data set with new
    data and examining the sliding window. Default is `5m`.
*   `custom-query`: Customized query following
    [MQL](https://cloud.google.com/monitoring/mql/reference) to use for query
    instead. By specifying this, the query will not be crafted by the program.
    The program will just ensure that the error condition has not been met for
    the trigger duration.

## 4. Create a release

Create a Cloud Deploy release for the configuration defined in the
`configuration` directory.

```shell
gcloud deploy releases create release-001 \
    --delivery-pipeline=my-verify-pipeline \
    --project=$PROJECT_ID \
    --region=$REGION \
    --source=configuration
```

[Open the Cloud Deploy UI](https://console.cloud.google.com/deploy), click on
the `my-verify-pipeline` pipeline, and then "Advance to stable". This pipeline
uses a canary strategy, which is why it needs to be advanced. It will then
advance and you can click on the release and then the rollout to see the verify
job.
