# Cloud Deploy Helm Deployer Sample
This directory contains a sample implementation of a Cloud Deploy Custom Target for deploying to a Google Kubernetes Engine (GKE) cluster with Helm.

**This is not an officially supported Google product, and it is not covered by a
Google Cloud support contract. To report bugs or request features in a Google
Cloud product, please contact [Google Cloud
support](https://cloud.google.com/support).**

# Quickstart
A quickstart that uses this sample is available [here](./quickstart/QUICKSTART.md)

# Configuration
The configuration provided when creating a Cloud Deploy Release must contain a [Helm chart](https://helm.sh/docs/topics/charts/).

# Deploy Parameters

| Parameter | Required | Description |
| --- | --- | --- |
| customTarget/helmGKECluster| Yes | Name of the GKE cluster the Helm chart is deployed to, e.g. `projects/{project}/locations/{location}/clusters/{cluster}` |
| customTarget/helmConfigurationPath | No | Path to the Helm chart in the Cloud Deploy release archive. If not provided then defaults to `mychart` in the root directory of the archive |
| customTarget/helmTemplateLookup | No | Whether to handle lookup functions when performing `helm template` for the informational release manifest, requires connecting to the cluster at render time |
| customTarget/helmTemplateValidate | No | Whether to validate the manifest produced by `helm template` against the cluster, requires connecting to the cluster at render time |
| customTarget/helmUpgradeTimeout | No | Timeout duration when performing `helm upgrade`, if unset relies on Helm default |

<a name="build"></a>
# Build the sample image and register a Custom Target Type for Helm
The `build_and_register.sh` script within this `helm` directory can be used to build the Helm deployer image and register a Cloud Deploy custom target type that references the image. To use the script run the following command:

```shell
./build_and_register.sh -p $PROJECT_ID -r $REGION
```

The script does the following on your behalf:
1. Create an Artifact Registry Repository
2. Give the default compute service account access to the Repository
3. Build the image and push it to the Repository
4. Create a Cloud Storage bucket and within the bucket a skaffold configuration that references the image built
5. Apply a custom target type for Helm to Cloud Deploy that references the skaffold configuration in Cloud Storage

# How the sample image works
The Helm deployer sample image is built to handle both a render and deploy request from Cloud Deploy.

## Render
The render process consists of the following steps:

1. Download the configuration provided at Release creation time and find the Helm chart based on the `customTarget/helmConfigurationPath` deploy parameter.

2. If either the `customTarget/helmTemplateLookup` or `customTarget/helmTemplateValidate` deploy parameter is set to `true` then get the cluster credentials.

3. Run `helm template` for the provided Helm chart using the Cloud Deploy Delivery Pipeline ID as the Helm Release name.

    a. If `customTarget/helmTemplateLookup` is `true` then `--dry-run=server` arg is used.

    b. If `customTarget/helmTemplateValidate` is `true` then `--validate` arg is used.

4. Upload to Cloud Storage the manifest produced by `helm template` to be used as the [Cloud Deploy Release inspector](https://cloud.google.com/deploy/docs/view-release#view_release_artifacts) artifact.

5. Upload the configuration to Cloud Storage so the Helm chart is available at deploy time.

## Deploy
The deploy process consists of the following steps:

1. Download the configuration that was uploaded to Cloud Storage during the render process.

2. Get the cluster credentials.

3. Run `helm upgrade` for the provided Helm chart using the Cloud Deploy Delivery Pipeline ID as the Helm Release name.

    a. If `customTarget/helmUpgradeTimeout` is set, e.g. `10m`, then `--timeout=10m` arg is used.

4. Run `helm get manifest` to get the manifest applied by the Helm Release and upload it to Cloud Storage as a Cloud Deploy deploy artifact.
