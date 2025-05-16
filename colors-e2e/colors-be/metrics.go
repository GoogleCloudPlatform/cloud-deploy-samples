package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"cloud.google.com/go/compute/metadata"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	googlepb "github.com/golang/protobuf/ptypes/timestamp"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	"google3/third_party/golang/protobuf/v1/proto/proto"
)

type ServiceMetadata struct {
	podName         string
	podNamespace    string
	clusterName     string
	clusterLocation string
	projectID       string
	releaseId       string
	deploymentName  string
}

func GetSerivceMetadata() (*ServiceMetadata, error) {
	clusterName, err := metadata.InstanceAttributeValue("cluster-name")
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster name: %w", err)
	}
	clusterLocation, err := metadata.InstanceAttributeValue("cluster-location")
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster location: %w", err)
	}

	projectId, err := metadata.ProjectID()
	if err != nil {
		return nil, fmt.Errorf("failed to get project id: %w", err)
	}

	podName := os.Getenv("PodName")
	if podName == "" {
		return nil, errors.New("PodName env var not set")
	}

	podNamespace := os.Getenv("PodNamespace")
	if podNamespace == "" {
		return nil, errors.New("podNamespace env var not set")
	}

	deploymentName := os.Getenv("DeploymentName")
	if deploymentName == "" {
		return nil, errors.New("DeploymentName env var not set")
	}

	releaseId := os.Getenv("ReleaseId")
	if releaseId == "" {
		return nil, errors.New("ReleaseId env var not set")
	}

	return &ServiceMetadata{
		podName:         podName,
		podNamespace:    podNamespace,
		clusterName:     clusterName,
		clusterLocation: clusterLocation,
		projectID:       projectId,
		releaseId:       releaseId,
		deploymentName:  deploymentName,
	}, nil
}

// Simple request logger for sample application
// Real applications should use something like OpenTelemetry
type RequestLogger struct {
	client          *monitoring.MetricClient
	serviceMetadata *ServiceMetadata
	goodRequests    int64
	badRequests     int64
	metricSendCount int64
	ctx             context.Context
}

func NewRequestLogger(ctx context.Context, serviceMetadata *ServiceMetadata) (*RequestLogger, error) {
	// Creates a client.
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric client: %w", err)
	}

	logger := &RequestLogger{
		client:          client,
		serviceMetadata: serviceMetadata,
		goodRequests:    0,
		badRequests:     0,
		metricSendCount: 0,
		ctx:             ctx,
	}

	// For sample application, just send collected metrics every 10 seconds
	go func() {
		for {
			logger.SendMetrics()
			time.Sleep(time.Second * 10)
			logger.metricSendCount++
			if logger.metricSendCount%6 == 0 {
				log.Printf("[Metric Heartbeat] Current sent sample count %v", logger.metricSendCount)
			}
		}
	}()

	return logger, nil
}

// Sends the current counts to monitoring and clears the counters
func (l *RequestLogger) SendMetrics() {
	var goodCount int64 = 0
	var badCount int64 = 0
	badCount = atomic.SwapInt64(&l.badRequests, badCount)
	goodCount = atomic.SwapInt64(&l.goodRequests, goodCount)
	request := &monitoringpb.CreateTimeSeriesRequest{
		Name: fmt.Sprintf("projects/%s", l.serviceMetadata.projectID),
		TimeSeries: []*monitoringpb.TimeSeries{
			l.MakeTimeSeriesWithDataPoint("2xx", goodCount),
			l.MakeTimeSeriesWithDataPoint("5xx", badCount),
		}}

	if err := l.client.CreateTimeSeries(l.ctx, request); err != nil {
		log.Printf("Failed to write time series data: %v\n", err)
		log.Printf("Request message: %v\n", proto.MarshalTextString(request))
	}
}

func (l *RequestLogger) LogRequest(ctx context.Context, isGood bool) {
	if isGood {
		atomic.AddInt64(&l.goodRequests, 1)
	} else {
		atomic.AddInt64(&l.badRequests, 1)
	}
}

func (l *RequestLogger) MakeTimeSeriesWithDataPoint(responseCodeClass string, metricValue int64) *monitoringpb.TimeSeries {
	dataPoint := &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime: &googlepb.Timestamp{
				Seconds: time.Now().Unix(),
			},
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_Int64Value{
				Int64Value: metricValue,
			},
		},
	}

	return &monitoringpb.TimeSeries{
		Metric: &metricpb.Metric{
			Type: "custom.googleapis.com/requests/request_count",
			Labels: map[string]string{
				"deployment_name":     l.serviceMetadata.deploymentName,
				"release_id":          l.serviceMetadata.releaseId,
				"response_code_class": responseCodeClass,
			},
		},
		Resource: &monitoredrespb.MonitoredResource{
			Type: "k8s_pod",
			Labels: map[string]string{
				"project_id":     l.serviceMetadata.projectID,
				"location":       l.serviceMetadata.clusterLocation,
				"cluster_name":   l.serviceMetadata.clusterName,
				"pod_name":       l.serviceMetadata.podName,
				"namespace_name": l.serviceMetadata.podNamespace,
			},
		},
		Points: []*monitoringpb.Point{
			dataPoint,
		},
	}
}
