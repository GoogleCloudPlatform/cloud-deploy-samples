# Cloud Deploy Vertex AI Deployer Sample
This directory contains a sample implementation of a Cloud Deploy Custom Target for deploying Vertex AI Models to an endpoint.

**This is not an officially supported Google product, and it is not covered by a
Google Cloud support contract. To report bugs or request features in a Google
Cloud product, please contact [Google Cloud
support](https://cloud.google.com/support).**

# Quickstart

A quickstart that uses this sample is available [here](./quickstart/QUICKSTART.md).

# Configuration


## Deploy Parameters

This custom deployer sample require certain [Deploy Parameters](https://cloud.google.com/deploy/docs/parameters) to be provided to function.

The table below lists the supported deploy parameters, whether the parameter is required, and the recommended resource where the parameter should be defined.

| Parameter               | Required | Recommended Location | Description                                                                                                                                                                  | 
|-------------------------|----------|----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| customTarget/vertexAIModel           | Yes      | Release              | Model to deploy. Format is "projects/{project}/locations/{location}/models/{modelId}"                                                                                        |
| customTarget/vertexAIMinReplicaCount | Yes      | Release              | The minimum replica count to assign for the deployed model.                                                                                                                  |
| customTarget/vertexAIEndpoint        | Yes      | Target               | The Vertex AI endpoint where the model will be deployed to. Format is "projects/{project}/locations/{location}/endpoints/{endpointId}"                                       |
| customTarget/vertexAIConfigurationPath      | No       | -                    | Path to the DeployedModel configuration in the Cloud Deploy Release archive. If not provided then defaults to file `deployedModel.yaml` in the root directory of the archive |
| customTarget/vertexAIAliases         | No       | Target               | Comma-separated list of aliases that should be assigned to a model after a deployment. Required when using the add alias option for the deployer.                            |

# Building the sample image
The `build_and_register.sh` script within this `model-deployer` directory can be used to build the Vertex AI model deployer image and register a Cloud Deploy custom target type that references the image. To use the script run the following command:

```shell
./build_and_register.sh -p $PROJECT_ID -r $REGION
```

The script does the following on your behalf:
1. Create an Artifact Registry Repository
2. Give the default compute service account access to the Repository
3. Build the image and push it to the Repository
4. Create a Cloud Storage bucket and within the bucket a skaffold configuration that references the image built
5. Apply a custom target type for Vertex AI to Cloud Deploy that references the skaffold configuration in Cloud Storage

# How the sample image works

The sample image is built to handle both a render, deploy, and post-deploy request from Cloud Deploy. It also supports canary deployment by splitting endpoint traffic
across multiple deployed models.

## Per-target configuration

This custom deployer deploys a model by calling the `projects.locations.endpoints.deployModel` [API method](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/deployModel).

The request takes as input a `DeployedModel` object.

The `DeployModelRequest` object passed as an argument is generated during the `Render` operation and stored as a YAML file in Google Cloud Storage.

You can provide the `DeployedModel` portion of the request by writing a file containing the YAML representation of the `DeployedModel` model object.

Example `deployedModel.yaml`:
```text
displayName:test_model
dedicatedResources:
  minReplicaCount: 3
  maxReplicaCount: 9
```

By default, the deployer will look for a `deployedModel.yaml` under the source folder directory (where the skaffold file is located). If found, the DeployedModel definition
within the file will be applied to all targets in the pipeline.

To specify a different location for the configuration, use the deploy parameter `customTarget/vertexAIConfigurationPath`.

For example, for the following `customTarget/vertexAIConfigurationPath` value:
```
customTarget/vertexAIConfigurationPath=prod
```

The deployer will look for  a configuration with the following path (relative to the directory containing the `skaffold.yaml`)

`prod/deployedModel.yaml`

See the [quickstart](./quickstart/QUICKSTART.md) for a practical example.
## Using placeholders in the configuration file

In your configuration file, you can add placeholders for any values you want to substitute with the value of deploy parameters. These values will be substituted
during the rendering step. See the [Cloud Deploy documentation](https://cloud.google.com/deploy/docs/parameters#add_placeholders) on deploy parameters for an explanation
on how this substitution works.

## Render

1. The configuration file `deployedModel.yaml` is loaded, the deploy parameter `customTarget/vertexAIConfigurationPath` determines the location if its provided.
2. Placeholders in the config file are set with the corresponding deploy parameter value if it exists.
3. The field minReplicaCount is set using the provided `customTarget/vertexAIMinReplicaCount` deploy parameter value if its not provided in a `deployedModel.yaml` file.
4. The model resource name passed using `customTarget/vertexAIModel` is adjusted to also include the model version ID (if it's not already provided) then this value is set in the request
5. If this is a canary deployment, the traffic split is generated to route traffic between the new model and previous model. Since actual deployment can occur much later than when the rendering of this manifest occurs,
   we use a placeholder for the previously deployed model, and resolve the ID of the previous model during deploy time.
6. The `DeployedModelRequest` object that is built is transformed into YAML and stored in Google Cloud Storage to be retrieved during deployment.

## Deploy

1. The `DeployedModelRequest` object is retrieved from Google Cloud Storage and parsed into a DeployedModelRequest object.
2. If its a canary deployment, the `previous-model` placeholder in the traffic split portion of the request is replaced with the ID of actual previous model.
3. The [deployModel](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/deployModel) API method is called, using deploy parameter value `customTarget/vertexAIEndpoint` to
   deploy to the desired endpoint.
4. Once the model deployment has completed, the Vertex AI endpoint is queried for all deployed models and any model with zero traffic is un-deployed.
