# Colors E2E Demo
This sample demonstrates an end-to-end CI/CD pipeline that leverages a large number of Cloud Deploy features , including:

  - Canary Deployments using pod-based traffic splitting
  - Integration with Cloud Monitoring to determine if deployments are successful using a verify job
  - Automated promotion
  - Setting up annotations based on build data
  - Parallel deployments
  - Deployment paramters 

## Application architecture

Colors is a simple application that contains a webpage that displays a stream of colors based on the configuration of the backend.

The demo application consists of two services: colors-be and colors-fe. 

###  Colors-be 
* Acts as a backend API that returns a configured color (this can be set via an environment variable).
* Logs metrics to Cloud Ops Suite to report the status of every API request that it serves
* To simulate user traffic, generates constant load to the API endpoint
* Reads an environment variable to inject faults in a percentage of its responses

### Colors-fe
* Acts as the front end to the colors application and renders a webpage that shows the history of colors returned from calling colors-be on a periodic basis.
* Also displays the value of select environment variables


## Setup

### Prerequisites
To setup the demo you will need:
* A GCP project 
* 4 GKE clusters representing dev, staging and two prod clusters
* An artifact registry repository setup to store the images needed for this demo.  GKE needs permission to read from this repository and Cloud Build (if using) or the user running the build needs permission to push images
* Cloud Monitoring APIs enabled

### Steps
1. Update the variables in ‘update_files_with_vars.sh’ and run the script to update the configuration files to match your setup.
2. Build the verify-metrics-container image at https://github.com/GoogleCloudPlatform/cloud-deploy-samples/tree/main/verify-evaluate-cloud-metrics and upload it to the image repository that you created
3. Apply  the Cloud Deploy configuration by running “gcloud deploy apply -f clouddeploy.yaml”
4. Deploy the backend by running “gcloud builds submit --config=colors-be/cloudbuild.yaml
5. Deploy the front end by running “gcloud builds submit –config=colors-fe/cloudbuild.yaml”

## Things to try

* Setup port forwarding to set the front end in your various clusters 
```
gcloud container clusters get-credentials <cluster> --zone <zone> --project <project>
kubectl port-forward service/colors-fd-scv 8080:8080 --context=gke_<project>_<zone>_<cluster>
```
You can do this multiple times with different local ports in order to view multiple clusters from the same machine at the same time 

* Update the override color in colors-be/k8s.yaml, trigger a deployment and see how the change propagates through environments, especially where canary is configured. Trigger a rollback via the UI to see that as well

* Change the deploy parameters in the colors-fd pipeline in clouddeploy.yaml

* Update the fault percentage deploy parameter in the colors-be pipeline in clouddeploy.yaml. Re-apply the file, trigger a deployment and see how the faults impact the deployment. Also look for the impact in the Cloud Monitoring dashboard 
