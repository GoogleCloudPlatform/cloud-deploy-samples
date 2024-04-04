# Cloud Deploy Infrastructure Manager Terraform Deployer Sample
This directory contains a sample implementation of a Cloud Deploy Custom Target for deploying Google
Cloud infrastructure resources via Terraform using [Infrastructure Manager](https://cloud.google.com/infrastructure-manager).

**This is not an officially supported Google product, and it is not covered by a
Google Cloud support contract. To report bugs or request features in a Google
Cloud product, please contact [Google Cloud
support](https://cloud.google.com/support).**

# Overview

The Infrastructure Manager deployer allows you to use Cloud Deploy to control the application of Terraform configurations with Infrastructure Manager via a Cloud Deploy Delivery Pipeline.

Example use cases:
* Manage your infrastructure changes from the same interface as the rest of your application delivery
* Control the progression of your infrastructure changes between multiple environments (dev, staging, production) either manually or via autmation.
* Run tests after changes have been applied to validate their success
* Use any other Cloud Deploy features alongside Infrastructure Manager: approvals, rollbacks, pre/post deployment hooks, etc.

# Quickstart

A quickstart that uses this sample is available [here](./quickstart/QUICKSTART.md).

# Configuration

## Terraform Configuration
The Terraform configuration provided when creating a Cloud Deploy Release must meet the [Infrastructure Manager
constraints](https://cloud.google.com/infrastructure-manager/docs/terraform#constraints_on_terraform_configurations).

> [!IMPORTANT] 
> To ensure consistent Cloud Deploy Rollouts for a Release - remotely sourced modules should be [selected by revision](https://developer.hashicorp.com/terraform/language/modules/sources#selecting-a-revision), e.g. for a module in a Git repo specify a commit using its SHA-1 hash.

## Deploy Parameters

| Parameter | Required | Description |
| --- | --- | --- |
| customTarget/imProject | Yes | The project ID for the Infrastructure Manager Deployment |
| customTarget/imLocation | Yes | The location for the Infrastructure Manager Deployment. See Infrastructure Manager supported locations [here](https://cloud.google.com/infrastructure-manager/docs/locations) |
| customTarget/imDeployment | Yes | The ID of the Infrastructure Manager Deployment responsible for managing the Terraform configuration |
| customTarget/imConfigurationPath | No | Path to the Terraform configuration in the Cloud Deploy release archive. If not provided then defaults to the root directory of the archive |
| customTarget/imVariablePath | No | Path to a Terraform variable definition (.tfvars) file relative to the Terraform configuration |
| customTarget/imServiceAccount | No | Service account Infrastructure Manager uses when actuating resources. If not provided then defaults to the service account provided by the Cloud Deploy workload context |
| customTarget/imWorkerPool | No | Worker Pool Infrastructure Manager uses when creating Cloud Builds. If not provided then defaults to the worker pool provided by the Cloud Deploy workload context |
| customTarget/imImportExistingResources | No | Whether Infrastructure Manager should automatically import existing resources into the Terraform state and continue actuation. Check Infrastructure Manager documentation for import supported resources |
| customTarget/imDisableCloudDeployLabels | No | Whether to disable the Cloud Deploy labels applied on the Infrastructure Manager Deployment resource |

Additionally, Terraform variables can be passed in via deploy parameters with the prefix `customTarget/imVar_` followed by the name of a declared variable. For example, `customTarget/imVar_foo=bar` will set the `foo` variable value to `bar`.

<a name="build"></a>
# Build the sample image and register a Custom Target Type for Infrastructure Manager
The `build_and_register.sh` script within this `infrastructure-manager` directory can be used to build the Infrastructure Manager deployer image and register a Cloud Deploy custom target type that references the image. To use the script run the following command:

```shell
./build_and_register.sh -p $PROJECT_ID -r $REGION
```

The script does the following on your behalf:
1. Create an Artifact Registry Repository
2. Give the default compute service account access to the Repository
3. Build the image and push it to the Repository
4. Create a Cloud Storage bucket and within the bucket a skaffold configuration that references the image built
5. Apply a custom target type for Infrastructure Manager to Cloud Deploy that references the skaffold configuration in Cloud Storage

# How the sample image works
The Infrastructure Manager deployer sample image is built to handle both a render and deploy request from Cloud Deploy.

## Render
The render process consists of the following steps:

1. Download the configuration provided at Release creation time and find the Terraform configuration directory based on the `customTarget/imConfigurationPath` deploy parameter.

2. Within the Terraform configuration directory - generate variable definitions file (`clouddeploy.auto.tfvars`) based on variables declared in the file at `customTarget/imVariablePath` deploy parameter and declared by the `customTarget/imVar_` prefixed deploy parameters. 

3. Archive the Terraform configuration into a zip file and upload it to Cloud Storage.

4. Generate a YAML representation of the Infrastructure Manager Deployment that will be applied at deploy time and upload to Cloud Storage. The Deployment contains the reference to the Terraform configuration uploaded in step (3). The Deployment YAML is viewable in the [Cloud Deploy Release inspector](https://cloud.google.com/deploy/docs/view-release#view_release_artifacts).

## Deploy
The deploy process consists of the following steps:

1. Download the Infrastructure Manager Deployment YAML that was uploaded during the render process.

2. Create or Update the Deployment and wait for Infrastructure Manager to finish applying the Terraform configuration.

3. Terraform output values are passed back to Cloud Deploy as metadata to be populated on the Rollout.
