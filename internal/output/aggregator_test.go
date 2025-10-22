package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResultAggregator_Add_And_Aggregate(t *testing.T) {
	agg := NewResultAggregator()

	// Add some hosts
	agg.Add(AggregatedHost{
		Name:     "host1",
		Status:   StatusSuccess,
		Duration: 100 * time.Millisecond,
	})

	agg.Add(AggregatedHost{
		Name:     "host2",
		Status:   StatusSuccess,
		Duration: 150 * time.Millisecond,
	})

	agg.Add(AggregatedHost{
		Name:         "host3",
		Status:       StatusFailed,
		Duration:     200 * time.Millisecond,
		ErrorMessage: "Connection refused",
	})

	// Aggregate
	results := agg.Aggregate()

	// Check grouping
	assert.Equal(t, 2, len(results))

	// First group should be successful hosts
	assert.Equal(t, StatusSuccess, results[0].Status)
	assert.Equal(t, 2, results[0].Count)
	assert.InDelta(t, 66.67, results[0].Percentage, 0.1)

	// Second group should be failed hosts
	assert.Equal(t, StatusFailed, results[1].Status)
	assert.Equal(t, 1, results[1].Count)
	assert.InDelta(t, 33.33, results[1].Percentage, 0.1)
}

func TestResultAggregator_GetMetrics(t *testing.T) {
	agg := NewResultAggregator()

	agg.Add(AggregatedHost{
		Name:     "host1",
		Status:   StatusSuccess,
		Duration: 100 * time.Millisecond,
	})

	agg.Add(AggregatedHost{
		Name:     "host2",
		Status:   StatusChanged,
		Changed:  true,
		Duration: 200 * time.Millisecond,
	})

	agg.Add(AggregatedHost{
		Name:     "host3",
		Status:   StatusFailed,
		Duration: 150 * time.Millisecond,
	})

	metrics := agg.GetMetrics()

	assert.Equal(t, 3, metrics.Total)
	assert.Equal(t, 1, metrics.SuccessCount)
	assert.Equal(t, 1, metrics.ChangedCount)
	assert.Equal(t, 1, metrics.FailedCount)
	assert.Equal(t, 450*time.Millisecond, metrics.TotalDuration)
	assert.Equal(t, 150*time.Millisecond, metrics.AverageDuration)
	assert.Equal(t, 100*time.Millisecond, metrics.FastestDuration)
	assert.Equal(t, 200*time.Millisecond, metrics.SlowestDuration)
}

func TestResultAggregator_Filter(t *testing.T) {
	agg := NewResultAggregator()

	agg.Add(AggregatedHost{
		Name:   "host1",
		Status: StatusSuccess,
	})

	agg.Add(AggregatedHost{
		Name:   "host2",
		Status: StatusFailed,
	})

	agg.Add(AggregatedHost{
		Name:   "host3",
		Status: StatusSuccess,
	})

	// Filter for successful hosts
	filtered := agg.Filter(StatusSuccess)
	assert.Equal(t, 2, filtered.total)

	// Filter for failed hosts
	filtered = agg.Filter(StatusFailed)
	assert.Equal(t, 1, filtered.total)
}

func TestResultAggregator_Sort(t *testing.T) {
	agg := NewResultAggregator()

	agg.Add(AggregatedHost{
		Name:     "charlie",
		Status:   StatusSuccess,
		Duration: 100 * time.Millisecond,
	})

	agg.Add(AggregatedHost{
		Name:     "alice",
		Status:   StatusSuccess,
		Duration: 300 * time.Millisecond,
	})

	agg.Add(AggregatedHost{
		Name:     "bob",
		Status:   StatusFailed,
		Duration: 200 * time.Millisecond,
	})

	// Sort by name
	agg.Sort("name")
	assert.Equal(t, "alice", agg.results[0].Name)
	assert.Equal(t, "bob", agg.results[1].Name)
	assert.Equal(t, "charlie", agg.results[2].Name)

	// Sort by duration
	agg.Sort("duration")
	assert.Equal(t, 300*time.Millisecond, agg.results[0].Duration)
	assert.Equal(t, 200*time.Millisecond, agg.results[1].Duration)
	assert.Equal(t, 100*time.Millisecond, agg.results[2].Duration)
}
