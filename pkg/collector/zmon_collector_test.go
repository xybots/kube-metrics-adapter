package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zalando-incubator/kube-metrics-adapter/pkg/zmon"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type zmonMock struct {
	dataPoints []zmon.DataPoint
	entities   []zmon.Entity
}

func (m zmonMock) Query(checkID int, key string, tags map[string]string, aggregators []string, duration time.Duration) ([]zmon.DataPoint, error) {
	return m.dataPoints, nil
}

func TestZMONCollectorNewCollector(t *testing.T) {
	collectPlugin, _ := NewZMONCollectorPlugin(zmonMock{})

	config := &MetricConfig{
		MetricTypeName: MetricTypeName{
			Metric: newMetricIdentifier(ZMONCheckMetric),
		},
		Config: map[string]string{
			zmonCheckIDLabelKey:             "1234",
			zmonAggregatorsLabelKey:         "max",
			zmonTagPrefixLabelKey + "alias": "cluster_alias",
			zmonDurationLabelKey:            "5m",
			zmonKeyLabelKey:                 "key",
		},
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}

	collector, err := collectPlugin.NewCollector(hpa, config, 1*time.Second)
	require.NoError(t, err)
	require.NotNil(t, collector)
	zmonCollector := collector.(*ZMONCollector)
	require.Equal(t, "key", zmonCollector.key)
	require.Equal(t, 1234, zmonCollector.checkID)
	require.Equal(t, 1*time.Second, zmonCollector.interval)
	require.Equal(t, 5*time.Minute, zmonCollector.duration)
	require.Equal(t, []string{"max"}, zmonCollector.aggregators)
	require.Equal(t, map[string]string{"alias": "cluster_alias"}, zmonCollector.tags)

	// should fail if the metric name isn't ZMON
	config.Metric = newMetricIdentifier("non-zmon-check")
	_, err = collectPlugin.NewCollector(nil, config, 1*time.Second)
	require.Error(t, err)

	// should fail if the check id is not specified.
	delete(config.Config, zmonCheckIDLabelKey)
	config.Metric.Name = ZMONCheckMetric
	_, err = collectPlugin.NewCollector(nil, config, 1*time.Second)
	require.Error(t, err)
}

func newMetricIdentifier(metricName string) autoscalingv2.MetricIdentifier {
	selector := metav1.LabelSelector{}
	return autoscalingv2.MetricIdentifier{Name: metricName, Selector: &selector}
}

func TestZMONCollectorGetMetrics(tt *testing.T) {

	config := &MetricConfig{
		MetricTypeName: MetricTypeName{
			Metric: newMetricIdentifier(ZMONCheckMetric),
			Type:   "foo",
		},
		Config: map[string]string{
			zmonCheckIDLabelKey:             "1234",
			zmonAggregatorsLabelKey:         "max",
			zmonTagPrefixLabelKey + "alias": "cluster_alias",
			zmonDurationLabelKey:            "5m",
			zmonKeyLabelKey:                 "key",
		},
	}

	for _, ti := range []struct {
		msg              string
		dataPoints       []zmon.DataPoint
		collectedMetrics []CollectedMetric
	}{
		{
			msg: "test successfully getting metrics",
			dataPoints: []zmon.DataPoint{
				{
					Time:  time.Time{},
					Value: 1.0,
				},
			},
			collectedMetrics: []CollectedMetric{
				{
					Type: config.Type,
					External: external_metrics.ExternalMetricValue{
						MetricName:   config.Metric.Name,
						MetricLabels: config.Metric.Selector.MatchLabels,
						Timestamp:    metav1.Time{Time: time.Time{}},
						Value:        *resource.NewMilliQuantity(int64(1.0)*1000, resource.DecimalSI),
					},
				},
			},
		},
		{
			msg: "test not getting any metrics",
		},
	} {
		tt.Run(ti.msg, func(t *testing.T) {
			z := zmonMock{
				dataPoints: ti.dataPoints,
			}

			zmonCollector, err := NewZMONCollector(z, config, 1*time.Second)
			require.NoError(t, err)

			metrics, _ := zmonCollector.GetMetrics()
			require.Equal(t, ti.collectedMetrics, metrics)
		})
	}
}

func TestZMONCollectorInterval(t *testing.T) {
	collector := ZMONCollector{interval: 1 * time.Second}
	require.Equal(t, 1*time.Second, collector.Interval())
}