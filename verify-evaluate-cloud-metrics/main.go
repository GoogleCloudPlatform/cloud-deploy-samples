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

// Package main contains the logic for using Cloud Monitoring to determine whether requests have been receiving 5xx errors.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
)

var (
	// Variable to hold the flag's values.
	project            string
	tableName          string
	metricType         string
	predicates         string
	responseCodeClass  string
	maxErrorPercentage float64
	triggerDuration    time.Duration
	timeToMonitor      time.Duration
	slidingWindow      time.Duration
	refreshPeriod      time.Duration

	// Custom Query. If this is specified, then the query will not be crafted by the program.
	customQuery string
)

func getQueryText(timeOfStart time.Time) string {
	if len(customQuery) != 0 {
		return customQuery
	}
	var sb strings.Builder
	// Fetch from the table name and the metric type specified by arguments.
	sb.WriteString(fmt.Sprintf("fetch %s::%s", tableName, metricType))
	// Include the predicates to filter on.
	parts := strings.Split(predicates, ",")
	if len(parts) > 0 {
		holder := ""
		for i, p := range parts {
			holder += p
			if i != len(parts)-1 {
				holder += " && "
			}
		}
		sb.WriteString(" | ")
		sb.WriteString(fmt.Sprintf("(%s)", holder))
	}
	// Specify the start time.
	sb.WriteString(" | ")
	duration := time.Since(timeOfStart)
	sb.WriteString(fmt.Sprintf("within d'%s'", duration.String()))
	// Group by the specified sliding window
	sb.WriteString(" | ")
	sb.WriteString(fmt.Sprintf("group_by sliding(%v)", slidingWindow))
	// Filter the error ratio.
	sb.WriteString(" | ")
	sb.WriteString(fmt.Sprintf("filter_ratio response_code_class == '%s'", responseCodeClass))

	return sb.String()
}

func formatMsg(in string) string {
	if len(customQuery) > 0 {
		return fmt.Sprintf("(ignore due to custom query) %s", in)
	}
	return in
}

// replaceEnvVars replaces env var refs in the string with their value (if set). Env var refs are made
// with the format $envVarName
func replaceEnvVars(input string) string {

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		input = strings.ReplaceAll(input, "$"+pair[0], pair[1])
	}
	return input
}

func init() {
	// Initializing of the flag and print out the values for visibility.
	flag.StringVar(&project, "project", os.Getenv("CLOUD_DEPLOY_PROJECT"), "The ID of the project that has the metrics defined, defaulted to the CLOUD_DEPLOY_PROJECT environmental variable")
	flag.StringVar(&tableName, "table-name", "", "The [tablename](https://cloud.google.com/monitoring/mql/reference#fetch-tabop) to fetch from")
	flag.StringVar(&metricType, "metric-type", "", "The [metric type](https://cloud.google.com/monitoring/mql/reference#metric-tabop) to get from the table-name")
	flag.StringVar(&predicates, "predicates", "", "Commma delimited list of [predicates](https://cloud.google.com/monitoring/mql/reference#filter-tabop)")
	flag.StringVar(&responseCodeClass, "response-code-class", "5xx", "The response_code_class that is being monitored for the error condition")
	flag.Float64Var(&maxErrorPercentage, "max-error-percentage", 10, "The maximum allowable percentage of the specified response_code_class per sliding window")
	flag.DurationVar(&slidingWindow, "sliding-window", time.Minute, "The duration of the sliding window")
	flag.DurationVar(&triggerDuration, "trigger-duration", 5*time.Minute, "The time required to observe the error condition for verify to fail")
	flag.DurationVar(&timeToMonitor, "time-to-monitor", 20*time.Minute, "The time to monitor for response failures before the verification is marked successful")
	flag.DurationVar(&refreshPeriod, "refresh-period", 5*time.Minute, "The time to wait before refreshing the data set with new data")
	flag.StringVar(&customQuery, "custom-query", "", "Customized query following [MQL](https://cloud.google.com/monitoring/mql/reference) to use for query instead. By specifying this, the query will not be crafted by the program")

	flag.Parse()
	project = replaceEnvVars(project)
	tableName = replaceEnvVars(tableName)
	metricType = replaceEnvVars(metricType)
	predicates = replaceEnvVars(predicates)
	responseCodeClass = replaceEnvVars(responseCodeClass)

	fmt.Println("---")
	fmt.Println("Verification configured as follows:")
	fmt.Printf("Project: %q\n", project)
	fmt.Println(formatMsg(fmt.Sprintf("Table Name: %q", tableName)))
	fmt.Println(formatMsg(fmt.Sprintf("Metric Type: %q", metricType)))
	fmt.Println(formatMsg(fmt.Sprintf("Predicates: %q", predicates)))
	fmt.Println(formatMsg(fmt.Sprintf("Response Code Class: %q", responseCodeClass)))
	fmt.Printf("Max Error Percentage: %v\n", maxErrorPercentage)
	fmt.Println(formatMsg(fmt.Sprintf("Sliding Window: %v", slidingWindow)))
	fmt.Printf("Trigger Duration: %v\n", triggerDuration)
	fmt.Printf("Time To Monitor: %v\n", timeToMonitor)
	fmt.Printf("Refresh Period: %v\n", refreshPeriod)
	fmt.Println("---")
}

func main() {
	if err := do(); err != nil {
		fmt.Printf("err: %v", err)
		os.Exit(1)
	}
	fmt.Println("Done")
}

func do() error {
	ctx := context.Background()
	client, err := monitoring.NewQueryClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create NewQueryClient: %w", err)
	}
	defer client.Close()

	timeToStart := time.Now()
	timeToEnd := timeToStart.Add(timeToMonitor)

	queryToUse := getQueryText(timeToStart)
	fmt.Printf("The query is %q\n", queryToUse)

	refreshCount := 1
	for time.Now().Before(timeToEnd) {
		triggered, err := errorConditionTriggered(ctx, client, refreshCount, queryToUse)
		if err != nil {
			return fmt.Errorf("failed to determine whether error condition triggered: %w", err)
		}
		if triggered {
			return fmt.Errorf("verify failed, error condition triggered for more than duration")
		}
		time.Sleep(refreshPeriod)
		refreshCount++
	}
	return nil
}

// Validates that the error condition was not exceeded for trigger_duration on the sliding window.
func errorConditionTriggered(ctx context.Context, client *monitoring.QueryClient, refreshCount int, query string) (bool, error) {
	req := &monitoringpb.QueryTimeSeriesRequest{
		Name:  fmt.Sprintf("projects/%s", project),
		Query: query,
	}

	it := client.QueryTimeSeries(ctx, req)
	fmt.Printf("querying the time series, refresh count: %d\n", refreshCount)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return false, fmt.Errorf("could not read time series value: %w", err)
		}
		// The sliding window calculation are based on the points of a singular time series.
		startTimeOfErrorCondition := time.Time{}
		endTimeOfErrorCondition := time.Time{}
		var dataPoints []*monitoringpb.TimeSeriesData_PointData
		for _, p := range resp.GetPointData() {
			errorRatio := p.GetValues()[0].GetDoubleValue() * 100
			fmt.Printf("error ratio: %f\n", errorRatio)
			fmt.Printf("Start time: %v\n", p.GetTimeInterval().StartTime.AsTime())
			fmt.Printf("End time: %v\n", p.GetTimeInterval().EndTime.AsTime())

			if calculateDuration(startTimeOfErrorCondition, endTimeOfErrorCondition) >= triggerDuration {
				// We check to see if the sliding windows that we have set from previous iterations exceed the trigger duration.
				// If it has, then we stop reading point data.
				break
			}
			// Time series list data points from newest data to oldest data.
			if len(p.GetValues()) != 1 {
				// Assuming that the point data is a ratio.
				return false, fmt.Errorf("expected 1 rate value for the total interval, instead got: %d", len(p.GetValues()))
			}

			if errorRatio := p.GetValues()[0].GetDoubleValue() * 100; errorRatio >= maxErrorPercentage {
				if endTimeOfErrorCondition.IsZero() {
					// initialization
					endTimeOfErrorCondition = p.GetTimeInterval().EndTime.AsTime()
				}
				// Always replace the start as we iterate; it gets earlier and earlier.
				dataPoints = append([]*monitoringpb.TimeSeriesData_PointData{p}, dataPoints...)
				startTimeOfErrorCondition = p.GetTimeInterval().StartTime.AsTime()
			} else {
				// We found a sliding window which does not violate percentage.
				startTimeOfErrorCondition = time.Time{}
				endTimeOfErrorCondition = time.Time{}
				dataPoints = nil // reset the points
			}
		}
		// We check to see if the sliding windows that we have set from previous iterations exceed the trigger duration.
		if errorDuration := calculateDuration(startTimeOfErrorCondition, endTimeOfErrorCondition); errorDuration >= triggerDuration {
			fmt.Printf("found duration in which max error percentage %f exceeded trigger duration, duration condition triggered for: %v\n", maxErrorPercentage, errorDuration)
			fmt.Printf("data: %v\n", dataPoints)
			return true, nil
		}
	}
	return false, nil
}

func calculateDuration(start time.Time, end time.Time) time.Duration {
	if start.IsZero() {
		return 0
	}
	if end.IsZero() {
		return 0
	}
	return end.Sub(start)
}
