# Cloud Deploy Kubernetes Clean Up Sample

This contains source code for a container that can be used to clean up
Kubernetes resources that were deployed by Cloud Deploy.

**This is not an officially supported Google product, and it is not covered by a
Google Cloud support contract. To report bugs or request features in a Google
Cloud product, please contact
[Google Cloud support](https://cloud.google.com/support).**

By default, when deploying to Kubernetes clusters, the deploy phase of Cloud
Deploy uses `kubectl apply` to send rendered manifests to the Kubernetes control
plane. This means the control plane only sees resources that are listed in the
manifest. If you remove or rename resources, the old resources will not get
removed from the cluster.

This postdeploy hook implements a solution to this by deleting resources that
were deployed by a previous release, but which aren't part of this release's
manifest.

At a high level, the sample image:

1.  Gets a list of kubernetes resources that were deployed by Cloud Deploy in
    the **current** release, filtered to the pipeline, target, project-id and
    location.
2.  Gets a list of all kubernetes resources that were deployed by Cloud Deploy
    on the cluster, filtered to the pipeline, target, project-id and location.
    This includes resources deployed by **any** release associated with the
    current pipeline.
3.  Does a diff and deletes any resources that were not deployed as part of the
    current release (i.e. deletes all the old resources).

## Configuration

There are two flags that control what resources are deleted. These can be passed
to the container via the `args` of the Skaffold custom action in the Skaffold
configuration. See
[the `skaffold.yaml` file from the quickstart](quickstart/configuration/skaffold.yaml#L22)
for an example.

### `--namespace`

This flag specifies a comma-separated list of namespaces that will be queried
when looking for resources. By default, it will query all namespaces.

### `--resource-type`

This flag specifies the list of resources to delete, and the order in which they
will be deleted.

By default, it deletes the following resource types (and in this order):

*   `service`
*   `cronjob.batch`
*   `job.batch`
*   `deployment.apps`
*   `statefulset.apps`
*   `pod`
*   `configmap`
*   `secret`
*   `horizontalpodautoscaler.autoscaling`

To delete all resource types, use `--resource-type=all`.

To make changes to the default (e.g., adding or removing resources), the
simplest thing is to [copy the default list from the source code](main.go#L17)
and add additional resources to it.

## Quickstart

A quickstart that uses this sample is available
[here](./quickstart/QUICKSTART.md).
