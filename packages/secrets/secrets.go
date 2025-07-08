// Package secrets contains utilities for accessing secrets from Secret Manager.
package secrets

import (
	"context"
	"fmt"
	"hash/crc32"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// SecretVersionData accesses the Secret Manager SecretVersion and returns the data payload.
func SecretVersionData(ctx context.Context, secretVersion string, smClient *secretmanager.Client) (string, error) {
	fmt.Printf("Accessing SecretVersion %s\n", secretVersion)
	res, err := smClient.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretVersion,
	})
	if err != nil {
		return "", fmt.Errorf("failed to access secret version %s: %v", secretVersion, err)
	}
	crc32c := crc32.MakeTable(crc32.Castagnoli)
	// Verify the data checksum
	checksum := int64(crc32.Checksum(res.Payload.Data, crc32c))
	if checksum != *res.Payload.DataCrc32C {
		return "", fmt.Errorf("data corruption detected with secret version")
	}
	fmt.Printf("Accessed SecretVersion %s\n", secretVersion)
	return string(res.Payload.Data), nil
}
