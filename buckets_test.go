package main

import (
	"fmt"
	"reverseproxy/trackers/buckets"
	"testing"
)

func assertSlicesEqual(t *testing.T, given, expected []int, message string) {
	t.Helper() // Marks this function as a helper to improve test failure output

	if len(given) != len(expected) {
		t.Errorf("%s: slices have different lengths, given: %d, expected: %d", message, len(given), len(expected))
		return
	}

	for i := range given {
		if given[i] != expected[i] {
			t.Errorf("%s: slices differ at index %d, given: %v, expected: %v", message, i, given, expected)
			return
		}
	}
}
func TestBuckets(t *testing.T) {
	bucketsDef := []float64{
		0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8, 25.6, 51.12, 102.4, 204.8,
	}
	stats := buckets.NewPerPathStats(bucketsDef)

	statsDemo := stats.GetStatsForPath("/demo")

	statsDemo.Record(1, 200)
	statsDemo.Record(0.5, 200)
	statsDemo.Record(50, 404)
	statsDemo.Record(90, 200)
	statsDemo.Record(100, 200)
	statsDemo.Record(110, 200)
	statsDemo.Record(150, 200)
	statsDemo.Record(151, 500)
	statsDemo.Record(250, 200)

	expectedBucketCounts := []int{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 2, 3, 0}

	assertSlicesEqual(t, statsDemo.BucketCounts(), expectedBucketCounts, "bucket counts")

	tests := []struct {
		percentile float64
		expected   float64
	}{
		{0.10, 0.8},
		{0.20, 0.8},
		{0.50, 0.8},
		{0.90, 0.8},
		{0.95, 0.8},
		{0.98, 0.8},
		{0.99, 0.8},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%f", tt.percentile), func(t *testing.T) {

			value := statsDemo.GetPercentile(tt.percentile)

			if value != tt.expected {
				t.Errorf("GetPercentile(%v) = %v, want %v", tt.percentile, value, tt.expected)
			}
		})
	}

}
