# Kubernetes Resource Clean Up
This contains a sample container that can be used to clean up Kubernetes
resources that were deployed by Cloud Deploy. It should be used as a [postdeploy
hook](https://cloud.google.com/deploy/docs/hooks) (a configuration example is provided below). By default, it deletes the 
following resource types and in this order, however this can be overridden with 
a command line flag:

* service
* cronjob.batch
* job.batch
* deployment.apps
* replicaset.apps
* statefulset.apps
* pod
* configmap
* secret
* horizontalpodautoscaler.autoscaling

At a high level, the sample image:
1. Gets a list of kubernetes resources that were deployed by Cloud Deploy in the 
   **current** release, filtered to the pipeline, target, project-id and
   location.
2. Gets a list of **all** kubernetes resources that were deployed by Cloud Deploy
   on the cluster, filtered to the pipeline, target, project-id and location.
3. Does a diff and deletes any resources that were not deployed as part of the
   current release (i.e. deletes all the old resources).

# Prerequisites
1. You will need to build the image and push it to a repository, accessible by
Cloud Build.
2. There is a sample clouddeploy.yaml, kubernetes.yaml and skaffold.yaml file in
the config-sample directory. Either use those and replace the PROJECT_ID, 
REGION, and IMAGE values or update your existing Cloud Deploy config
file to reference a postdeploy hook, and your Skaffold file to then reference
the image you built.
3. If using the samples, create a GKE cluster (note this can take 10 min to run):

```
gcloud container clusters create-auto cleanup-prod --project=PROJECT_ID --region=REGION
```
4. Do not disable cloud deploy labels via an org policy. If you have an org
policy set that disables labels, this won’t work. This is because kubernetes 
query uses the Cloud Deploy labels to filter to resources that were deployed
by Cloud Deploy.

Lastly, a quick note that if you have this postdeploy job configured then you
should provide all resources in your manifests when creating a release, even if
there's no change to prevent deletions. 

# Building and pushing the image to a repo
1. In the directory of this QUICKSTART, run the following command to build the image:

```
docker build --tag <REPO-TAG> . 
```

For example, if you're pushing to an Artifact Registry with:
* region=us-central1
* project=my-project
* docker repo=my-repo
* you'd like to name the image clean-up-image

The command would look like this:

```
docker build --tag us-central1-docker.pkg.dev/my-project/my-repo/clean-up-image .
```

2. After the build is complete, push the image to the repository:

```
docker push <REPO-PATH>
```

Sticking with the example above, the command would be:

```
docker push us-central1-docker.pkg.dev/my-project/my-repo/clean-up-image
```

# Update your config or use the sample configs

If you're using the sample config, go to the `config-samples` folder, and replace
the PROJECT_ID and REGION in the clouddeploy.yaml file. Replace the IMAGE in the
skaffold.yaml file with the image you built from this code. Save the three
config files. 

An overview
of the configuration files:
1. `clouddeploy.yaml`: Defines a Delivery Pipeline that references a single
[GKE Target](https://cloud.google.com/deploy/docs/deploy-app-gke) and specifies
a postdeploy action `cleanup-action`.
1. `kubernetes.yaml`: Defines an Deployment and Service that will be applied to the cluster.
1. `skaffold.yaml`: Defines a custom action `cleanup-action` which is referenced in the clouddeploy.yaml. 
Within that customAction stanza there is a reference to the image that was
built above. 

If you're updating your own configuration files, update your clouddeploy.yaml
to reference a postdeploy hook action and your skaffold.yaml to define that
custom action.

# Register your pipeline and target with Cloud Deploy

This assumes your clouddeploy.yaml is in the same directory, if not update the 
`--file` arg to point to the full path.

```
gcloud deploy apply --file=clouddeploy.yaml --region=REGION --project=PROJECT_ID
```

# Create a release and at the end the postdeploy hook will run

Create a release and after the release has been deployed to the cluster, the
postdeploy hook will run. If you're using the sample, the command would look 
something like the below command. 

```
gcloud deploy releases create my-release --project=PROJECT_ID --region=REGION --delivery-pipeline=mypipeline --images=my-app-image=gcr.io/google-containers/nginx@sha256:f49a843c290594dcf4d193535d1f4ba8af7d56cea2cf79d1e9554f077f1e7aaa
```

The `--images=` flag replaces the placeholder (my-app-image) in the kubernetes 
manifest with the specific, SHA-qualified image. In the case of the samples, 
an nginx container.

Now create another release, so that the postdeploy hook will actually do 
something and delete resources from the previous release `my-release`:

```
gcloud deploy releases create my-release2 --project=PROJECT_ID --region=REGION --delivery-pipeline=mypipeline --images=my-app-image=gcr.io/google-containers/nginx@sha256:f49a843c290594dcf4d193535d1f4ba8af7d56cea2cf79d1e9554f077f1e7aaa
```

# Additional configuration options

There are two command line flags you can pass to the container:

1. `namespace`: Namespace(s) to filter on when finding resources to delete. For 
    multiple namespaces, separate them with a comma (e.g. `namespace=foo,bar`).
    The default is to delete across all namespaces.
2. `resource-type`: Comma separated list of resource type(s) to filter on when finding resources to
    delete. If you want to add a few more to the default list, copy and paste
    the following, and add your own:
    "service,cronjob.batch,job.batch,deployment.apps,replicaset.apps,statefulset.apps,pod,configmap,secret,horizontalpodautoscaler.autoscaling"/
    Note that order is preserved - the order of the list is the order in which
    resources will be deleted. If you want to delete ALL resources, pass in
    "all" (i.e. `resource-type=all`)

Add the args in your skaffold config file in the customActions.containers
stanza. See the sample skaffold.yaml file for an example that's commented out.