package buckets

import (
	"math"
	"reverseproxy/trackers"
	"sync"
	"time"
)

type BucketStats struct {
	buckets       []float64 // Upper bounds for buckets
	bucketCounts  []int     // Counts for each bucket
	totalCalls    int       // Total calls
	totalTime     float64
	mutex         sync.Mutex
	StatusesCount map[int]int
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
}

// NewBucketStats initializes a BucketStats instance with the given bucket bounds.
func NewBucketStats(bucketBounds []float64) *BucketStats {
	return &BucketStats{
		buckets:       bucketBounds,
		bucketCounts:  make([]int, len(bucketBounds)+1), // +1 for overflow bucket
		totalTime:     0.0,
		StatusesCount: make(map[int]int),
		FirstSeen:     time.Now(),
	}
}

func (bs *BucketStats) Buckets() []float64 {
	return bs.buckets
}

func (bs *BucketStats) BucketCounts() []int {
	return bs.bucketCounts
}

func (bs *BucketStats) TotalCount() int64 {
	return trackers.SumArray(bs.bucketCounts)
}

func (bs *BucketStats) TotalTime() float64 {
	return bs.totalTime
}

// Record adds a response time to the appropriate bucket.
func (bs *BucketStats) Record(duration float64, statusCode int) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.totalTime += duration
	bs.LastSeen = time.Now()
	bs.totalCalls++
	bs.StatusesCount[statusCode]++
	for i, upperBound := range bs.buckets {
		if duration <= upperBound {
			bs.bucketCounts[i]++
			return
		}
	}

}

// GetPercentile computes the approximate value for the given percentile (e.g., 50, 95).
func (bs *BucketStats) GetPercentile(targetPercentile float64) float64 {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if bs.totalCalls == 0 {
		return 0 // No data recorded
	}

	threshold := int(math.Ceil(targetPercentile / 100.0 * float64(bs.totalCalls)))
	cumulative := 0

	for i, count := range bs.bucketCounts {
		cumulative += count
		if cumulative >= threshold {
			if i < len(bs.buckets) {
				return bs.buckets[i]
			}
			return bs.buckets[len(bs.buckets)-1] + 1 // Overflow bucket
		}
	}

	return 0 // Fallback
}

// PerPathStats holds bucket stats for each path
type PerPathStats struct {
	buckets []float64 // Upper bounds for buckets
	stats   map[string]*BucketStats
	mutex   sync.Mutex
}

// NewPerPathStats initializes a PerPathStats instance
func NewPerPathStats(bucketBounds []float64) *PerPathStats {
	return &PerPathStats{
		buckets: bucketBounds,
		stats:   make(map[string]*BucketStats),
	}
}

// GetStatsForPath returns the stats for a specific path, creating it if necessary
func (pps *PerPathStats) GetStatsForPath(path string) *BucketStats {
	pps.mutex.Lock()
	defer pps.mutex.Unlock()

	if _, exists := pps.stats[path]; !exists {
		pps.stats[path] = NewBucketStats(pps.buckets)
	}
	return pps.stats[path]
}

func (pps *PerPathStats) GetAllPercentiles() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	for path, stats := range pps.stats {
		percentiles := make(map[string]interface{})
		percentiles["50"] = stats.GetPercentile(50)
		percentiles["90"] = stats.GetPercentile(90)
		percentiles["95"] = stats.GetPercentile(95)
		percentiles["98"] = stats.GetPercentile(98)
		percentiles["99"] = stats.GetPercentile(99)

		percentiles["counts"] = stats.bucketCounts

		percentiles["totalTime"] = stats.totalTime
		percentiles["totalCount"] = trackers.SumArray(stats.bucketCounts)
		percentiles["statusCount"] = stats.StatusesCount

		percentiles["firstSeen"] = stats.FirstSeen
		percentiles["lastSeen"] = stats.LastSeen

		result[path] = percentiles
	}

	return result
}
