# Cloud Deploy Git Deployer Quickstart

This page shows you how to use Cloud Deploy to deploy to a Git Repository on `github.com` using the sample Git deployer.

In this quickstart you will:

1. Create a GitHub repository and a Personal Access Token to give a service account access to the repository.
2. Define a Cloud Deploy delivery pipeline, custom target type for Git, and two targets (dev and prod).
3. Create a Cloud Deploy release and rollout to write the rendered manifest to the Git repository under a `dev` path in the `deploy` branch and create a pull request to the `main` branch.
4. Promote the release to deploy the rendered manifest to the Git repository under a `prod` path in the `deploy` branch and create a pull request to the `main` branch.

## 1. Clone Repository

Clone this repository and navigate to the quickstart directory (`cloud-deploy-samples/custom-targets/git-ops/quickstart`) since the commands provided expect to be executed from that directory.

## 2. Environment Variables

To simplify the commands in this quickstart, set the following environment variables with your values:

```shell
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="YOUR_REGION"
```

## 3. Prerequisites

### APIs
Enable the Cloud Deploy API and Secret Manager API.

```shell
gcloud services enable clouddeploy.googleapis.com secretmanager.googleapis.com container.googleapis.com --project $PROJECT_ID
```

### Permissions
Make sure the default compute service account, `{project_number}-compute@developer.gserviceaccoumt.com`, used by Cloud Deploy has sufficient permissions:

1. `clouddeploy.jobRunner` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/clouddeploy.jobRunner"
    ```

2. `secretmanager.secretAccessor` role:

    ```shell
    gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$(gcloud projects describe $PROJECT_ID \
    --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
    --role="roles/secretmanager.secretAccessor"
    ```

## 4. Build and Register a Custom Target Type for Git
From within the `quickstart` directory, run the following command to build the Git deployer image and register a Cloud Deploy custom target type that references the image:

```shell
../build_and_register.sh -p $PROJECT_ID -r $REGION
```

For information about the `build_and_register.sh` script, see the [README](../README.md#build)

## 5. Create GitHub Repository and Personal Access Token (PAT)

Create a [new GitHub repository](https://github.com/new) with a README since the quickstart depends on the existence of a `main` branch. The rendered manifests will be written to this repository during the deployment process. Set the repository and owner as environment variables:

```shell
export GIT_REPO="YOUR_REPO" # e.g. my-test-repo
export GIT_OWNER="YOUR_OWNER" # e.g. my-username
```

Then create a [new PAT](https://github.com/settings/personal-access-tokens/new) for the repository and under the `Repository permissions` section grant `Read and Write` access for `Contents` and `Pull requests`. Copy the PAT for later use.

## 6. Create a Secret Manager Secret Version for the PAT

Create the secret:

```shell
gcloud secrets create git-deployer-ct-quickstart-pat --replication-policy="automatic" --project=$PROJECT_ID
```

> [!WARNING]
> The next two steps to create the secret version will print the secret in plaintext. Do not do this if this secret has access to a non-test repository. Instead reference a file path, steps [here](https://cloud.google.com/secret-manager/docs/add-secret-version#secretmanager-add-secret-version-gcloud).

Set the PAT as an environment variable for easy use in later step:

```shell
export GIT_PAT="YOUR_PAT"
```

Add a secret version with the PAT:

```shell
echo -n $GIT_PAT | \
gcloud secrets versions add git-deployer-ct-quickstart-pat --data-file=- --project=$PROJECT_ID
```

Add the secret and version as environment variables:

```shell
export SECRET_ID=git-deployer-ct-quickstart-pat
export SECRET_VERSION=1
```

## 7. Create delivery pipeline and targets
Replace the placeholders in the `clouddeploy.yaml` in this directory with your environment variable values:

```shell
sed -i "s/\$PROJECT_ID/${PROJECT_ID}/g" ./clouddeploy.yaml
sed -i "s/\$GIT_OWNER/${GIT_OWNER}/g" ./clouddeploy.yaml
sed -i "s/\$GIT_REPO/${GIT_REPO}/g" ./clouddeploy.yaml
sed -i "s/\$SECRET_ID/${SECRET_ID}/g" ./clouddeploy.yaml
sed -i "s/\$SECRET_VERSION/${SECRET_VERSION}/g" ./clouddeploy.yaml
```

Apply the Cloud Deploy configuration:

```shell
gcloud deploy apply --file=clouddeploy.yaml --project=$PROJECT_ID --region=$REGION
```

## 8. Create a release
Create a Cloud Deploy release for the configuration defined in the `configuration` directory. This will automatically create a rollout that deploys to the Git repository and path configured for the dev target.

```shell
gcloud deploy releases create release-001 --delivery-pipeline=git-pipeline --project=$PROJECT_ID --region=$REGION --source=configuration
```

## 9. Check rollout status for dev target
In the [Cloud Deploy UI](https://console.cloud.google.com/deploy/delivery-pipelines) for your project click on the `git-pipeline` delivery pipeline. Here you can see the release created and the rollout to the dev target for the release.

You can also describe the rollout created using the following command:

```shell
gcloud deploy rollouts describe release-001-to-git-dev-0001 --release=release-001 --delivery-pipeline=git-pipeline --project=$PROJECT_ID --region=$REGION
```

Once the rollout has succeeded there should be a pull request created in the Git repository from branch `deploy` to `main` with the rendered manifest in the path `dev/k8s.yaml`. These were all configured as deploy parameters, see the `clouddeploy.yaml`.

## 10. Promote the release
Promote the release to start a rollout for the prod target.

```shell
gcloud deploy releases promote --release=release-001 --delivery-pipeline=git-pipeline --project=$PROJECT_ID --region=$REGION
```

## 11. Check the rollout status for prod target
View the `git-pipeline` delivery pipeline in the [Cloud Deploy UI](https://console.cloud.google.com/deploy/delivery-pipelines).

To describe the prod rollout run the following command:

```shell
gcloud deploy rollouts describe release-001-to-git-prod-0001 --release=release-001 --delivery-pipeline=git-pipeline --project=$PROJECT_ID --region=$REGION
```

Once the rollout has succeeded there should be a pull request created in the Git repository from branch `deploy` to `main` with the rendered manifest in the path `prod/k8s.yaml`.

## 12. Clean up

Delete the Secret Manager secret:

```shell
gcloud secrets delete git-deployer-ct-quickstart-pat --project=$PROJECT_ID
```

Delete the Cloud Deploy resources:

```shell
gcloud deploy delete --file=clouddeploy.yaml --force --project=$PROJECT_ID --region=$REGION
```