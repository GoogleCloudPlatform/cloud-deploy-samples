# Verify through Monitoring
This contains a sample verify container that can be used with Cloud Deploy during Deployment Verification. This app ensures that a certain response class for requests do not exceed a percentage threshold for a given amount of time. It uses [MQL](https://cloud.google.com/monitoring/mql) and builds up a query to send to the monitoring API.

Within the `cloud-deploy` folder, there are sample YAMLs.
1. `clouddeploy.yaml`: Defines a single [Cloud Run Target](https://cloud.google.com/deploy/docs/deploy-app-run). Defines a Delivery Pipeline that references that Cloud Run Target and specifies [Automated Canary Strategy](https://cloud.google.com/deploy/docs/deployment-strategies/canary).
1. `run.yaml`: Defines the service to be deployed to the Target.
1. `skaffold.yaml`: Defines the deployer, associated manifests, and the container configuration for verification.

# Prerequisites 
1. You will need to build the image and push it to a repository, accessible by Cloud Build.
1. You will need to set up a Cloud Run target with deployment permissions granted to the default compute account inside a project with Cloud Deploy enabled.
1. In the `clouddeploy.yaml` and the `skaffold.yaml`, you will need to replace the %PROJECT_ID%, %RUN_LOCATION%, and %IMAGE% values with ones appropriate for your project, target location, and container image.

# Building and pushing the image to a repo
1. In the directory of this README, run the following command to build the image:

```
docker build . -t <REPO-TAG>
```

1. After completing the build, push the image to the repository:

```
docker push <REPO-PATH>
```

# Running the example
You can configure the following inputs within the `skaffold.yaml`:
* `project`: the project to look for the metrics. This defaults to the env variable: `CLOUD_DEPLOY_PROJECT`. More environment variables can be viewed [here](https://cloud.google.com/deploy/docs/verify-deployment#available_environment_variables).
* `table-name`: the monitoring [tablename](https://cloud.google.com/monitoring/mql/reference#fetch-tabop) to fetch from.
* `metric-type`: The [metric type](https://cloud.google.com/monitoring/mql/reference#metric-tabop) to get from the table-name.
* `predicates`: Commma delimited list of [predicates](https://cloud.google.com/monitoring/mql/reference#filter-tabop) to be applied in the query
* `response-code-class`: The response_code_class to monitor for the error condition. Default is `5xx`.
* `max-error-percentage`: The maximum allowable percentage of the specified response_code_class in a sliding window. Default is `10`.
* `sliding-window`: The duration of the sliding window during the query. Default is `1m`. 
* `trigger-duration`: The duration required to observe the error condition for verify to fail. Default is `5m`. 
* `time-to-monitor`: The time to run this verification container for. If the time-to-monitor expires and there are no error conditions that has lasted >= the length of the trigger duration, this verification is marked as successful. Default is `20m`.
* `refresh-period`: The time to wait before refreshing the data set with new data and examining the sliding window. Default is `5m`.
* `custom-query`: Customized query following [MQL](https://cloud.google.com/monitoring/mql/reference) to use for query instead. By specifying this, the query will not be crafted by the program. The program will just ensure that the error condition has not been met for the trigger duration.
