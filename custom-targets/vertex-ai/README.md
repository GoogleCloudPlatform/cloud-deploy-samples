# Cloud Deploy Vertex AI Deployer Sample
This directory contains a sample implementation of a Cloud Deploy Custom Target for deploying Vertex AI Models to an endpoint.

**This is not an officially supported Google product, and it is not covered by a
Google Cloud support contract. To report bugs or request features in a Google
Cloud product, please contact [Google Cloud
support](https://cloud.google.com/support).**

# Quickstart

A quickstart that uses this sample is available [here](./quickstart/QUICKSTART.md).

# Configuration

## Deployed Model YAML
The Vertex AI model deployer expects a YAML representation of a [DeployedModel](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints#DeployedModel) to be provided when a Cloud Deploy Release is created. The deployer supports substituting placeholder values in the DeployedModel YAML with values provided as [deploy parameters](https://cloud.google.com/deploy/docs/parameters)

```text
displayName:test_model
dedicatedResources:
  minReplicaCount: 3
  maxReplicaCount: 9
```

## Deploy Parameters

This custom deployer sample require certain [Deploy Parameters](https://cloud.google.com/deploy/docs/parameters) to be provided to function.

The table below lists the supported deploy parameters, whether the parameter is required, and the recommended resource where the parameter should be defined.

| Parameter                              | Required | Recommended Location | Description                                                                                                                                                                   | 
|----------------------------------------|----------|----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| customTarget/vertexAIModel             | Yes      | Release              | Model to deploy. Format is "projects/{project}/locations/{location}/models/{modelId}".                                                                                        |
| customTarget/vertexAIMinReplicaCount   | No       | Target               | The minimum replica count to assign for the deployed model. This deploy parameter is required if its not provided in the `DeployedModel` YAML configuration.                  |
| customTarget/vertexAIEndpoint          | Yes      | Target               | The Vertex AI endpoint where the model will be deployed to. Format is "projects/{project}/locations/{location}/endpoints/{endpointId}"                                        |
| customTarget/vertexAIConfigurationPath | No       | -                    | Path to the DeployedModel configuration in the Cloud Deploy Release archive. If not provided then defaults to file `deployedModel.yaml` in the root directory of the archive. |
| customTarget/vertexAIAliases           | No       | Target               | Comma-separated list of aliases that should be assigned to a model after a deployment. Required when using the add alias option for the deployer.                             |

# Building the sample image
The `build_and_register.sh` script within this `vertex-ai` directory can be used to build the Vertex AI model deployer image and register a Cloud Deploy custom target type that references the image. To use the script run the following command:

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

The Vertex AI model deployer sample image is built to handle Cloud Deploy render and deploy requests including canary configurations via endpoint traffic splitting between the new model and the previously deployed model.

In addition, this image can be used in a [Cloud Deploy post-deployment hook](https://cloud.google.com/deploy/docs/hooks) to assign aliases to the model after it has been successfully deployed to an endpoint.

## Using placeholders in the configuration file

In your configuration file, you can add placeholders for any values you want to substitute with the value of deploy parameters. These values will be substituted
during the rendering step. See the [Cloud Deploy documentation](https://cloud.google.com/deploy/docs/parameters#add_placeholders) on deploy parameters for an explanation
on how this substitution works.

## Render

1. Download the configuration provided at Release creation time and locate the `DeployedModel` YAML file based on the deploy parameter `customTarget/vertexAIConfigurationPath`. (The default is already documented above)
2. Placeholders in the `DeployedModel` YAML are substituted with the set deploy parameters
3. The field minReplicaCount is set using the provided `customTarget/vertexAIMinReplicaCount` deploy parameter value if its not provided in a `deployedModel.yaml` file.
4. The model resource name passed using `customTarget/vertexAIModel` is adjusted to also include the model version ID (if it's not already provided) then this value is set in the request
5. If this is a canary deployment, the traffic split is generated to route traffic between the new model and previous model. Since actual deployment can occur much later than when the rendering of this manifest occurs,
   we use a placeholder for the previously deployed model, and resolve the ID of the previous model during deploy time.
6. A [Deploy Model Request Body](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/deployModel) is constructed based on the `DeployedModel` YAML and the generated traffic split. It's then uploaded to Google Cloud Storage to be used at deploy time.
   The request body is also viewable in the [Cloud Deploy release inspector](https://cloud.google.com/deploy/docs/view-release#view_release_artifacts)

## Deploy

1. Download the [Deploy Model Request Body](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/deployModel) that was uploaded during the render process.
2. If its a canary deployment, the `previous-model` placeholder in the traffic split portion of the request is replaced with the ID of actual previous model.
3. The [deployModel](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/deployModel) API method is called, using deploy parameter value `customTarget/vertexAIEndpoint` to
   deploy to the desired endpoint.
4. Once the model deployment has completed, the Vertex AI endpoint is queried for all deployed models and any model with zero traffic is un-deployed.


## Assigning aliases using a post-deploy hook

The custom image supports adding aliases to the deployed Vertex AI models, this functionality is meant to be
invoked through a post-deploy hook. The post-deploy runs the custom image, and provides the `--add-aliases-mode` flag to activate this 
functionality.

Additional configuration for the Delivery Pipeline and `skaffold.yaml` provided to the release is needed to activate this feature.

See the [quickstart](./quickstart/QUICKSTART.md) for an example.
