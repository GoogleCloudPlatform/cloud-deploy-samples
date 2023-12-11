package main

import (
	"context"
	"fmt"
	"google.golang.org/api/aiplatform/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

const (
	// wait for 30 seconds for a response regarding an operation.
	lroOperationTimeout = 30 * time.Second
	// Polling duration, regardless of how long the lease is, we're going to poll for at most 30 mins.
	pollingTimeout = 30 * time.Minute
)

// Poll will return the status of an operation if it finished within "operationTimeout" or an error
// indicating that the operation is incomplete.
func Poll(ctx context.Context, service *aiplatform.Service, op *aiplatform.GoogleLongrunningOperation) error {

	opService := aiplatform.NewProjectsLocationsOperationsService(service)

	_, err := opService.Get(op.Name).Do()

	if err != nil {
		return fmt.Errorf("unable to get operation")
	}

	pollFunc := GetWaitFunc(opService, op.Name, ctx)

	err = wait.PollUntilContextTimeout(ctx, lroOperationTimeout, pollingTimeout, true, pollFunc)

	if err != nil {
		return err
	}
	return nil
}

// GetWaitFunc waits for stuff
func GetWaitFunc(service *aiplatform.ProjectsLocationsOperationsService, name string, ctx context.Context) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (done bool, err error) {

		op, err := service.Get(name).Do()

		if err != nil {
			return false, err
		}

		if op.Done {
			return true, nil
		}

		return false, nil

	}
}

func pollChan(ctx context.Context, service *aiplatform.Service, lros ...*aiplatform.GoogleLongrunningOperation) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)
	wg.Add(len(lros))

	output := func(lro *aiplatform.GoogleLongrunningOperation) {
		out <- Poll(ctx, service, lro)
		wg.Done()
	}

	for _, lro := range lros {
		go output(lro)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
