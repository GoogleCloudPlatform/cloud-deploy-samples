// Copyright 2023 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// poll will return the status of an operation if it finished within "operationTimeout" or an error
// indicating that the operation is incomplete.
func poll(ctx context.Context, service *aiplatform.Service, op *aiplatform.GoogleLongrunningOperation) error {

	opService := aiplatform.NewProjectsLocationsOperationsService(service)

	_, err := opService.Get(op.Name).Do()

	if err != nil {
		return fmt.Errorf("unable to get operation")
	}

	pollFunc := getWaitFunc(opService, op.Name, ctx)

	err = wait.PollUntilContextTimeout(ctx, lroOperationTimeout, pollingTimeout, true, pollFunc)

	if err != nil {
		return err
	}
	return nil
}

// getWaitFunc is a helper function that returns true if the specified operation has completed.
func getWaitFunc(service *aiplatform.ProjectsLocationsOperationsService, name string, ctx context.Context) wait.ConditionWithContextFunc {
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

// pollChan is a helper function that facilitates polling multiple long running operations in parallel
func pollChan(ctx context.Context, service *aiplatform.Service, lros ...*aiplatform.GoogleLongrunningOperation) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)
	wg.Add(len(lros))

	output := func(lro *aiplatform.GoogleLongrunningOperation) {
		out <- poll(ctx, service, lro)
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
