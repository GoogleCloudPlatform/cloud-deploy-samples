package params

import (
	"fmt"
	"os"
	"strings"
)

// cloudDeployPrefix is the prefix for environment variables containing information about the deployment
const cloudDeployPrefix = "CLOUD_DEPLOY_"

// cloudDeployCustomTargetPrefix is the prefix for deploy parameters that are supported or required by the custom target.
const cloudDeployCustomTargetPrefix = "CLOUD_DEPLOY_customTarget_"

// transformAndValidateEnvkey checks if the environment variable is a valid deploy parameter
// and transforms the environment variable key back to the original format.
func transformAndValidateEnvkey(key string) (bool, string) {
	if strings.HasPrefix(key, cloudDeployCustomTargetPrefix) {
		transformedKey := strings.TrimPrefix(key, cloudDeployCustomTargetPrefix)
		transformedKey = fmt.Sprintf("customTarget/%s", transformedKey)
		return true, transformedKey
	} else if strings.HasPrefix(key, cloudDeployPrefix) {
		return false, ""
	} else {
		return true, key
	}
}

// FetchCloudDeployParameters returns a  map of all environment variables and keys
// that can be used in template parametrization.
func FetchCloudDeployParameters() map[string]string {
	params := map[string]string{}
	environs := os.Environ()
	for _, environ := range environs {
		segments := strings.Split(environ, "=")
		if validKey, transformedKey := transformAndValidateEnvkey(segments[0]); validKey {
			params[transformedKey] = segments[1]
		}
	}
	return params
}
