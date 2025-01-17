# Cloud Deploy Kubernetes Clean Up Sample

This contains a sample container that can be used to clean up Kubernetes
resources that were deployed by Cloud Deploy. It should be used as a post-deploy
hook. By default, it deletes the following resource types and in this order,
however the resource types to delete can be overridden with configuration (see
quickstart).

*   service
*   cronjob.batch
*   job.batch
*   deployment.apps
*   replicaset.apps
*   statefulset.apps
*   pod
*   configmap
*   secret
*   horizontalpodautoscaler.autoscaling

**This is not an officially supported Google product, and it is not covered by a
Google Cloud support contract. To report bugs or request features in a Google
Cloud product, please contact
[Google Cloud support](https://cloud.google.com/support).**

# Quickstart

A quickstart that uses this sample is available
[here](./quickstart/QUICKSTART.md)
