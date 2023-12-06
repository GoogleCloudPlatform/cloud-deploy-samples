# Cloud Deploy Terraform Deployer Sample
This directory contains a sample implementation of a Cloud Deploy Custom Target for deploying Google
Cloud infrastructure resources using Terraform.

# Quickstart

A quickstart that uses this sample is available [here](./quickstart/QUICKSTART.md).

# Configuration

## Terraform Configuration
The Terraform configuration provided when creating a Cloud Deploy Release **cannot** have a backend
configured. The sample image will create a backend configuration file (`backend.tf`) in the Terraform root module based
on the required deploy parameters provided, see section below.

## Deploy Parameters

| Parameter | Required | Description | 
| --- | --- | --- |
|customTarget/tfBackendBucket| Yes | Name of the Cloud Storage bucket used to store the Terraform state |
|customTarget/tfBackendPrefix| Yes | Prefix to use for the Cloud Storage objects that represent the Terraform state |
|customTarget/tfConfigurationPath| No | Path to the Terraform configuration in the Cloud Deploy Release archive. If not provided then defaults to the root directory of the archive |
|customTarget/tfVariablePath| No | Path to a Terraform variable definition (.tfvars) file relative to the Terraform configuration |
|customTarget/tfEnableRenderPlan| No | Whether to generate a Terraform plan at render time for informational purposes, i.e. provide in the Cloud Deploy Release inspector. This plan is not used when deploying the configuration |
|customTarget/tfLockTimeout| No | Duration to retry a state lock, when unset Terraform defaults to 0s |
|customTarget/tfApplyParallelism| No | Parallelism to set when performing terraform apply, when unset Terraform defaults to 10 |

Additionally, Terraform variables can be passed in via deploy parameters with the prefix `TF_VAR_` followed by the name of a declared variable. For example, `TF_VAR_foo=bar` will set the `foo` variable value to `bar`.

<a name="build"></a>
# Build the sample image and register a Custom Target Type for Terraform
The `build_and_register.sh` script within this `terraform` directory can be used to build the Terraform deployer image and register a Cloud Deploy custom target type that references the image. To use the script run the following command:

```shell
./build_and_register.sh -p $PROJECT_ID -r $REGION
```

The script does the following on your behalf:
1. Create an Artifact Registry Repository
2. Give the default compute service account access to the Repository
3. Build the image and push it to the Repository
4. Create a Cloud Storage bucket and within the bucket a skaffold configuration that references the image built
5. Apply a custom target type for Terraform to Cloud Deploy that references the skaffold configuration in Cloud Storage

# How the sample image works
The Terraform deployer sample image is built to handle both a render and deploy request from Cloud Deploy.

## Render
The render process consists of the following steps:

1. Download the configuration provided at Release creation time and find the Terraform working directory based on the `customTarget/tfConfigurationPath` deploy parameter.

2. Within the Terraform working directory:

    a. Generate backend configuration file (`backend.tf`) based on the `customTarget/tfBackendBucket` and `customTarget/tfBackendPrefix` deploy parameters.

    b. Generate variable definitions file (`clouddeploy.auto.tfvars`) based on the variables declared in the file at `customTarget/tfVariablePath` deploy parameter and defined by the `TF_VAR_` prefixed deploy parameters.

    c. Initialize the working directory containing the Terraform configuration and validate it.

3. Generate a Cloud Deploy Release inspector artifact that contains the variables in `clouddeploy.auto.tfvars` and upload it to Cloud Storage.
    
    * If deploy parameter `customTarget/tfEnableRenderPlan` is set to `true` then this artifact will also contain a speculative Terraform plan for informational purposes. This plan is **not** used when applying the Terraform configuration at deploy time.

4. Archive the configuration and upload it to Cloud Storage to be used at deploy time.

## Deploy
The deploy process consists of the following steps:

1. Download the configuration that was uploaded during the render process.

2. Apply the Terraform configuration within the Terraform working directory, based on the `customTarget/tfConfigurationPath` deploy parameter. 

> [!NOTE]
> The Terraform configuration is not initialized because it was done during the render process. Initializing at render time ensures that multiple deploys will use the same versions of child modules in the case that any child modules were stored remotely (e.g. on Github).

3. Get the Terraform state and upload it to Cloud Storage as a Cloud Deploy Deploy Artifact.

4. Terraform output values are passed back to Cloud Deploy as metadata to be populated in the Rollout.
