// Package main contains the logic for using Cloud Monitoring to determine whether requests have been receiving 5xx errors.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
)

var (
	// Variable to hold the flag's values.
	project            string
	metricFilter       string
	maxErrorPercentage float64
	triggerDuration    time.Duration
	timeToMonitor      time.Duration
	samplingPeriod     time.Duration
	samplingWindow     time.Duration
)

func init() {
	// Initializing of the flag and print out the values for visibility.
	flag.StringVar(&project, "project", "", "The ID of the project that has the metrics")
	flag.StringVar(&metricFilter, "metric-filter", "", "A [monitoring filter](https://cloud.google.com/monitoring/api/v3/filters) that specifies which time series should be returned")
	flag.Float64Var(&maxErrorPercentage, "max-error-percentage", 10, "The maximum allowable percentage of 5xx response_code_class per sampling")
	flag.DurationVar(&triggerDuration, "trigger-duration", 5*time.Minute, "The time required to observe the error condition for verify to fail")
	flag.DurationVar(&timeToMonitor, "time-to-monitor", 20*time.Minute, "The time to monitor for response failures before the verification is marked successful")
	flag.DurationVar(&samplingPeriod, "sampling-period", time.Minute, "The time to wait in between each sampling")
	flag.DurationVar(&samplingWindow, "sampling-window", 5*time.Minute, "The window of time that specifies the dataset for each sampling")
	flag.Parse()

	fmt.Println("---")
	fmt.Println("Verification configured as follows:")
	fmt.Printf("Project: %q\n", project)
	fmt.Printf("Metric Filter: %q\n", metricFilter)
	fmt.Printf("Max Error Percentage: %v\n", maxErrorPercentage)
	fmt.Printf("Trigger Duration: %v\n", triggerDuration)
	fmt.Printf("Time To Monitor: %v\n", timeToMonitor)
	fmt.Printf("Sampling Period: %v\n", samplingPeriod)
	fmt.Printf("Sampling Window: %v\n", samplingWindow)
	fmt.Println("---")
}

const (
	responseLabelName    = "response_code_class"
	responseCodeClass5xx = "5xx"
)

func main() {
	ctx := context.Background()

	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		fmt.Printf("Unable to create NewMetricClient: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	timeToEnd := time.Now().Add(timeToMonitor)
	sampleIndex := 1
	var timeWhenMaxErrorPercentageWasExceeded time.Time

	for time.Now().Before(timeToEnd) {
		endTime := time.Now()
		// We look back the window amount.
		startTime := endTime.Add(-samplingWindow)

		res, err := metricsAreWithinThresholdForPeriod(ctx, client, sampleIndex, startTime, endTime)
		if err != nil {
			fmt.Printf("failed to read time series: %v\n", err)
			os.Exit(1)
		}
		if res {
			// Reset the timeWhenMaxErrorPercentageWasExceeded since we found a period where the metrics were within threshold.
			timeWhenMaxErrorPercentageWasExceeded = time.Time{}
		} else {
			fmt.Printf("Sampling Set %d has exceeded the max error percentage\n", sampleIndex)
			if timeWhenMaxErrorPercentageWasExceeded.IsZero() {
				// If this is the first time seeing the error condition, set the time to the end time of this period.
				timeWhenMaxErrorPercentageWasExceeded = endTime
			}
			if errorDuration := endTime.Sub(timeWhenMaxErrorPercentageWasExceeded); errorDuration >= triggerDuration {
				// The time in which the error exceeded is equal to or greater than the trigger duration.
				fmt.Printf("max error percentage has been exceeded for %v, verification will fail\n", errorDuration)
				os.Exit(1)
			}

		}
		sampleIndex++
		time.Sleep(samplingPeriod)
	}
	fmt.Printf("Done\n")
	os.Exit(0)
}

// Reads the time series within the specified time period and returns true/false on whether the maxErrorPercentage has been exceeded for this time period.
func metricsAreWithinThresholdForPeriod(ctx context.Context, client *monitoring.MetricClient, sampleIndex int, startTime, endTime time.Time) (bool, error) {
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   fmt.Sprintf("projects/%s", project),
		Filter: metricFilter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamp.Timestamp{
				Seconds: startTime.Unix(),
			},
			EndTime: &timestamp.Timestamp{
				Seconds: endTime.Unix(),
			},
		},
	}

	var responseClass5xxReq int64
	var totalReq int64

	it := client.ListTimeSeries(ctx, req)
	// Read and accumulate the values of requests for the time window specified by StartTime and EndTime. Then we compare the % of the requests with the maxErrorPercentage.
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return false, fmt.Errorf("could not read time series value: %w", err)
		}
		labels := resp.GetMetric().GetLabels()
		for _, p := range resp.GetPoints() {
			totalReq += p.GetValue().GetInt64Value()
			if r, ok := labels[responseLabelName]; ok && r == responseCodeClass5xx {
				responseClass5xxReq += p.GetValue().GetInt64Value()
			}
		}
	}

	fmt.Printf("Sampling Set: %d. Total Requests: %d, Response Class 5xx: %d\n", sampleIndex, totalReq, responseClass5xxReq)

	if totalReq == 0 {
		// If there aren't any requests, then it is fine.
		return true, nil
	}

	asPercentage := float64(responseClass5xxReq) / float64(totalReq) * 100

	return asPercentage < maxErrorPercentage, nil
}
