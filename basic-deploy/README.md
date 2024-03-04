# Basic Deploy
This is a basic deployment example that shows how to configure a Cloud Deploy pipeline with two phases and deploy the same manifest point to an existing image to both.

# Prerequisites 
You will need two GKE clusters with deployment permissions granted to the default compute account inside a project with Cloud Deploy enabled.

In the Cloud Deploy YAML you will need to replace the %PROJECT_ID%, %CLUSTER_LOCATION%, and %CLUSTER_NAME% values with ones appropriate for your clusters.

Configure gcloud to use your desired project and region by default

# Running the example

The clouddeploy.yaml can be applied by running:
```
  gcloud deploy apply –file=clouddeploy.yaml 
```

You can create your first release with
```
   gcloud deploy releases create ‘r$DATE$TIME’ –-delivery-pipeline=my-app-pipeline
```
You can then inspect the pipeline in the Cloud Console UI and promote the release to the second target 
