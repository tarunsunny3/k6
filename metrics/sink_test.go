package metrics

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounterSink(t *testing.T) {
	samples10 := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 100.0}
	now := time.Now()

	t.Run("add", func(t *testing.T) {
		t.Run("one value", func(t *testing.T) {
			sink := CounterSink{}
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 1.0, Time: now})
			assert.Equal(t, 1.0, sink.Value)
			assert.Equal(t, now, sink.First)
		})
		t.Run("values", func(t *testing.T) {
			sink := CounterSink{}
			for _, s := range samples10 {
				sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s, Time: now})
			}
			assert.Equal(t, 145.0, sink.Value)
			assert.Equal(t, now, sink.First)
		})
	})
	t.Run("calc", func(t *testing.T) {
		sink := CounterSink{}
		assert.Equal(t, 0.0, sink.Value)
		assert.Equal(t, time.Time{}, sink.First)
	})
	t.Run("format", func(t *testing.T) {
		sink := CounterSink{}
		for _, s := range samples10 {
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s, Time: now})
		}
		assert.Equal(t, map[string]float64{"count": 145, "rate": 145.0}, sink.Format(1*time.Second))
	})
}

func TestGaugeSink(t *testing.T) {
	samples6 := []float64{1.0, 2.0, 3.0, 4.0, 10.0, 5.0}

	t.Run("add", func(t *testing.T) {
		t.Run("one value", func(t *testing.T) {
			sink := GaugeSink{}
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 1.0})
			assert.Equal(t, 1.0, sink.Value)
			assert.Equal(t, 1.0, sink.Min)
			assert.Equal(t, true, sink.minSet)
			assert.Equal(t, 1.0, sink.Max)
		})
		t.Run("values", func(t *testing.T) {
			sink := GaugeSink{}
			for _, s := range samples6 {
				sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s})
			}
			assert.Equal(t, 5.0, sink.Value)
			assert.Equal(t, 1.0, sink.Min)
			assert.Equal(t, true, sink.minSet)
			assert.Equal(t, 10.0, sink.Max)
		})
	})
	t.Run("calc", func(t *testing.T) {
		sink := GaugeSink{}
		assert.Equal(t, 0.0, sink.Value)
		assert.Equal(t, 0.0, sink.Min)
		assert.Equal(t, false, sink.minSet)
		assert.Equal(t, 0.0, sink.Max)
	})
	t.Run("format", func(t *testing.T) {
		sink := GaugeSink{}
		for _, s := range samples6 {
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s})
		}
		assert.Equal(t, map[string]float64{"value": 5.0}, sink.Format(0))
	})
}

func TestTrendSink(t *testing.T) {
	unsortedSamples10 := []float64{0.0, 100.0, 30.0, 80.0, 70.0, 60.0, 50.0, 40.0, 90.0, 20.0}

	t.Run("add", func(t *testing.T) {
		t.Run("one value", func(t *testing.T) {
			sink := TrendSink{}
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 7.0})
			assert.Equal(t, uint64(1), sink.Count)
			assert.Equal(t, false, sink.sorted)
			assert.Equal(t, 7.0, sink.Min)
			assert.Equal(t, 7.0, sink.Max)
			assert.Equal(t, 7.0, sink.Avg)

			// The median value needs to be explicitly calculated
			// using the the `TrendSink.Calc` method.
			assert.Equal(t, 0.0, sink.Med)
		})
		t.Run("values", func(t *testing.T) {
			sink := TrendSink{}
			for _, s := range unsortedSamples10 {
				sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s})
			}
			assert.Equal(t, uint64(len(unsortedSamples10)), sink.Count)
			assert.Equal(t, false, sink.sorted)
			assert.Equal(t, 0.0, sink.Min)
			assert.Equal(t, 100.0, sink.Max)
			assert.Equal(t, 54.0, sink.Avg)

			// The median value needs to be explicitly calculated
			// using the the `TrendSink.Calc` method.
			assert.Equal(t, 0.0, sink.Med)
		})
	})

	tolerance := 0.000001
	t.Run("percentile", func(t *testing.T) {
		t.Run("no values", func(t *testing.T) {
			sink := TrendSink{}
			for i := 1; i <= 100; i++ {
				assert.Equal(t, 0.0, sink.P(float64(i)/100.0))
			}
		})
		t.Run("one value", func(t *testing.T) {
			sink := TrendSink{}
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 10.0})
			for i := 1; i <= 100; i++ {
				assert.Equal(t, 10.0, sink.P(float64(i)/100.0))
			}
		})
		t.Run("two values", func(t *testing.T) {
			sink := TrendSink{}
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 5.0})
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 10.0})
			assert.Equal(t, 5.0, sink.P(0.0))
			assert.Equal(t, 7.5, sink.P(0.5))
			assert.Equal(t, 5+(10-5)*0.95, sink.P(0.95))
			assert.Equal(t, 5+(10-5)*0.99, sink.P(0.99))
			assert.Equal(t, 10.0, sink.P(1.0))
		})
		t.Run("more than 2", func(t *testing.T) {
			sink := TrendSink{}
			for _, s := range unsortedSamples10 {
				sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s})
			}
			assert.InDelta(t, 0.0, sink.P(0.0), tolerance)
			assert.InDelta(t, 55.0, sink.P(0.5), tolerance)
			assert.InDelta(t, 95.5, sink.P(0.95), tolerance)
			assert.InDelta(t, 99.1, sink.P(0.99), tolerance)
			assert.InDelta(t, 100.0, sink.P(1.0), tolerance)
		})
	})
	t.Run("format", func(t *testing.T) {
		sink := TrendSink{}
		for _, s := range unsortedSamples10 {
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s})
		}
		expected := map[string]float64{
			"min":   0.0,
			"max":   100.0,
			"avg":   54.0,
			"med":   55.0,
			"p(90)": 91.0,
			"p(95)": 95.5,
		}
		result := sink.Format(0)
		require.Equal(t, len(expected), len(result))
		for k, expV := range expected {
			assert.Contains(t, result, k)
			assert.InDelta(t, expV, result[k], tolerance)
		}
	})
}

func BenchmarkTrendSink(b *testing.B) {
	// Produce a fixed random values set for the benchmark.
	values := make([]float64, 0, 10001)
	for i := 0; i < 10000; i++ {
		values = append(values, rand.Float64()*100)
	}

	b.Run("P", func(b *testing.B) {
		// Prepare a sample size table to control the
		// variation of the number of samples for each
		// benchmark.
		var table = []struct {
			percentile float64
			sinkSize   int
		}{
			// 50th percentile
			// Note that computing the 50th percentile is equivalent
			// to computing the median.
			{percentile: 0.5, sinkSize: 10},
			{percentile: 0.5, sinkSize: 100},
			{percentile: 0.5, sinkSize: 1000},
			{percentile: 0.5, sinkSize: 10000},

			// 90th percentile
			{percentile: 0.9, sinkSize: 10},
			{percentile: 0.9, sinkSize: 100},
			{percentile: 0.9, sinkSize: 1000},
			{percentile: 0.9, sinkSize: 10000},

			// 95th percentile
			{percentile: 0.95, sinkSize: 10},
			{percentile: 0.95, sinkSize: 100},
			{percentile: 0.95, sinkSize: 1000},
			{percentile: 0.95, sinkSize: 10000},

			// 99th percentile
			{percentile: 0.99, sinkSize: 10},
			{percentile: 0.99, sinkSize: 100},
			{percentile: 0.99, sinkSize: 1000},
			{percentile: 0.99, sinkSize: 10000},
		}

		for _, v := range table {
			b.Run(fmt.Sprintf("%d_with_%d_values", int(v.percentile*100), v.sinkSize), func(b *testing.B) {
				sink := TrendSink{Values: values[0:v.sinkSize]}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					sink.P(v.percentile)
				}
			})
		}
	})
}

func TestRateSink(t *testing.T) {
	samples6 := []float64{1.0, 0.0, 1.0, 0.0, 0.0, 1.0}

	t.Run("add", func(t *testing.T) {
		t.Run("one true", func(t *testing.T) {
			sink := RateSink{}
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 1.0})
			assert.Equal(t, int64(1), sink.Total)
			assert.Equal(t, int64(1), sink.Trues)
		})
		t.Run("one false", func(t *testing.T) {
			sink := RateSink{}
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: 0.0})
			assert.Equal(t, int64(1), sink.Total)
			assert.Equal(t, int64(0), sink.Trues)
		})
		t.Run("values", func(t *testing.T) {
			sink := RateSink{}
			for _, s := range samples6 {
				sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s})
			}
			assert.Equal(t, int64(6), sink.Total)
			assert.Equal(t, int64(3), sink.Trues)
		})
	})
	t.Run("calc", func(t *testing.T) {
		sink := RateSink{}
		assert.Equal(t, int64(0), sink.Total)
		assert.Equal(t, int64(0), sink.Trues)
	})
	t.Run("format", func(t *testing.T) {
		sink := RateSink{}
		for _, s := range samples6 {
			sink.Add(Sample{TimeSeries: TimeSeries{Metric: &Metric{}}, Value: s})
		}
		assert.Equal(t, map[string]float64{"rate": 0.5}, sink.Format(0))
	})
}

func TestDummySinkAddPanics(t *testing.T) {
	assert.Panics(t, func() {
		DummySink{}.Add(Sample{})
	})
}

func TestDummySinkCalcDoesNothing(t *testing.T) {
	sink := DummySink{"a": 1}
	sink.Calc()
	assert.Equal(t, 1.0, sink["a"])
}

func TestDummySinkFormatReturnsItself(t *testing.T) {
	assert.Equal(t, map[string]float64{"a": 1}, DummySink{"a": 1}.Format(0))
}
