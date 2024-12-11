package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

const (
	cloudDeployPrefix = "deploy.cloud.google.com/"
	releaseEnvKey     = "CLOUD_DEPLOY_RELEASE"
	projectEnvKey     = "CLOUD_DEPLOY_PROJECT_ID"
	locationEnvKey    = "CLOUD_DEPLOY_LOCATION"
	pipelineEnvKey    = "CLOUD_DEPLOY_DELIVERY_PIPELINE"
	targetEnvKey      = "CLOUD_DEPLOY_TARGET"
	outputFlag        = "-o"
	nameArg           = "name"
)

// resourcesToDelete returns a list of resources that are not in the current set of resources
// (i.e. the set of resources that were just deployed by Cloud Deploy in the most recent release).
func (ce CommandExecutor) resourcesToDelete(namespace, resourceTypeFlag string) ([]string, error) {
	// Step 1. Get a list of resource types to query.
	resourceTypes, err := ce.resourceTypesToQuery(resourceTypeFlag)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of resources types to query, err: %w", err)
	}

	// Step 2. Get a list of all resources on the cluster that were deployed by Cloud Deploy.
	allResources, err := ce.listResources(false, namespace, resourceTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of resources on the cluster, err: %w", err)
	}

	// Step 3. Get a list of resources that were deployed by Cloud Deploy as part of the latest
	// release on the cluster.
	currentResources, err := ce.listResources(true, namespace, resourceTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of current resources on the cluster, err: %w", err)
	}

	// Step 4. Do a diff to determine what resources were not deployed in the latest release and
	// should therefore be deleted.
	return diffSlices(allResources, currentResources), nil
}

// apiResourceQueryArgs returns the args to pass to kubectl to get a list of supported resource
// types on the cluster.
func apiResourcesQueryArgs() []string {
	return []string{
		"api-resources",
		"--verbs=list",
		outputFlag,
		nameArg,
	}
}

// kubectlGetArgs returns the args to pass to kubectl to get the resource name,
// given the resource type and namespace
func kubectlGetArgs(includeReleaseLabel bool, resourceType string, nspace string) []string {

	var labels []string
	if includeReleaseLabel {
		labels = append(labels, fmt.Sprintf("%srelease-id=%s", cloudDeployPrefix, os.Getenv(releaseEnvKey)))
	}
	labels = append(labels, fmt.Sprintf("%sdelivery-pipeline-id=%s", cloudDeployPrefix, os.Getenv(pipelineEnvKey)))
	labels = append(labels, fmt.Sprintf("%starget-id=%s", cloudDeployPrefix, os.Getenv(targetEnvKey)))
	labels = append(labels, fmt.Sprintf("%slocation=%s", cloudDeployPrefix, os.Getenv(locationEnvKey)))
	labels = append(labels, fmt.Sprintf("%sproject-id=%s", cloudDeployPrefix, os.Getenv(projectEnvKey)))

	labelsFormatted := strings.Join(labels, ",")
	labelArg := fmt.Sprintf("-l %s", labelsFormatted)
	args := []string{
		"get",
		outputFlag,
		nameArg,
		labelArg,
	}
	if nspace != "" {
		args = append(args, fmt.Sprintf("--namespace=%s", nspace))
	}
	args = append(args, resourceType)
	return args
}

// resourceTypesToQuery returns a list of resource types to query based on the command line flag value.
func (ce CommandExecutor) resourceTypesToQuery(resourceType string) ([]string, error) {
	var resourceTypes []string
	// If resourceType(s) were specified on the command line, use those. Otherwise
	// get the list of supported resource types on the cluster.
	if resourceType != "" {
		resourceTypes = strings.Split(resourceType, ",")
	} else {
		apiResourcesArgs := apiResourcesQueryArgs()
		output, err := ce.execCommand(apiResourcesArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to execute kubectl api-resources command: %w", err)
		}
		outputSplit := strings.Split(output, "\n")
		// Delete the empty line at the end
		resourceTypes = slices.DeleteFunc(outputSplit, isEmpty)
	}
	return resourceTypes, nil
}

// listResources returns a list of resources that were deployed by Cloud Deploy,
// filtered by resource type and namespace. If includeReleaseLabel is true, the
// resutls are further filtered to resources that were deployed by the current
// release.
func (ce CommandExecutor) listResources(includeReleaseLabel bool, namespaces string, resourceTypes []string) ([]string, error) {

	var resources []string
	for _, r := range resourceTypes {
		res, err := ce.resourcesPerType(includeReleaseLabel, namespaces, r)
		if err != nil {
			return nil, fmt.Errorf("attempting to get resource type \"%v\" resulted in err: %w", r, err)
		}
		resources = append(resources, res...)
	}
	return resources, nil
}

// resourcesPerType returns a list of resources per type.
func (ce CommandExecutor) resourcesPerType(includeReleaseLabel bool, namespaces string, resourceType string) ([]string, error) {
	var resources []string
	// Multiple namespaces could have been specified in the command line arg, split and loop through each.
	nspaces := strings.Split(namespaces, ",")
	for _, n := range nspaces {
		args := kubectlGetArgs(includeReleaseLabel, resourceType, n)
		output, err := ce.execCommand(args)
		if err != nil {
			return nil, fmt.Errorf("attempting to get resource type \"%v\" resulted in err: %w", resourceType, err)
		}
		if output != "" {
			// Separate out by line break and delete the empty line at the end.
			outputSplit := strings.Split(output, "\n")
			outputSplit = slices.DeleteFunc(outputSplit, isEmpty)
			resources = append(resources, outputSplit...)
		}
	}
	return resources, nil

}

// deleteResources deletes the given resources.
func (ce CommandExecutor) deleteResources(resources []string) error {
	fmt.Printf("Beginning to delete resources, there are %d resources to delete\n", len(resources))
	for _, resource := range resources {
		args := []string{"delete", resource, "--ignore-not-found=true"}
		_, err := ce.execCommand(args)
		if err != nil {
			return fmt.Errorf("attempting to delete resource %v resulted in err: %w", resource, err)
		}
	}
	return nil
}

func isEmpty(e string) bool {
	return e == ""
}
