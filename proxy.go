package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// Embed the entire "static" folder.
//
//go:embed static/*
var staticFiles embed.FS

type BucketStats struct {
	buckets      []float64 // Upper bounds for buckets
	bucketCounts []int     // Counts for each bucket
	totalCalls   int       // Total calls
	mutex        sync.Mutex
}

// NewBucketStats initializes a BucketStats instance with the given bucket bounds.
func NewBucketStats(bucketBounds []float64) *BucketStats {
	return &BucketStats{
		buckets:      bucketBounds,
		bucketCounts: make([]int, len(bucketBounds)+1), // +1 for overflow bucket
	}
}

// Record adds a response time to the appropriate bucket.
func (bs *BucketStats) Record(duration float64) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	for i, upperBound := range bs.buckets {
		if duration <= upperBound {
			bs.bucketCounts[i]++
			bs.totalCalls++
			return
		}
	}
	// Increment overflow bucket
	bs.bucketCounts[len(bs.bucketCounts)-1]++
	bs.totalCalls++
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

		result[path] = percentiles
	}

	return result
}

type IPTracker struct {
	mu               sync.Mutex
	hits             map[string]int
	banned           map[string]time.Time
	statusCountPerIp map[string]map[int]int
	threshold        int
	banDuration      time.Duration
}

func NewIPTracker(threshold int, banDuration time.Duration) *IPTracker {
	return &IPTracker{
		hits:             make(map[string]int),
		banned:           make(map[string]time.Time),
		statusCountPerIp: make(map[string]map[int]int),
		threshold:        threshold,
		banDuration:      banDuration,
	}
}

func (t *IPTracker) CheckBan(ip string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if banTime, banned := t.banned[ip]; banned {
		if time.Since(banTime) > t.banDuration {
			delete(t.banned, ip) // Unban IP after duration
		} else {
			return true // Still banned
		}
	}
	return false
}

func (t *IPTracker) IncrementHit(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.hits[ip]++
	if t.hits[ip] > t.threshold {
		t.banned[ip] = time.Now()
		delete(t.hits, ip) // Reset count after banning
		log.Printf("Banned IP: %s", ip)
	}
}

func (t *IPTracker) IncrementStatus(ip string, statusCode int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, exists := t.statusCountPerIp[ip]

	if !exists {
		t.statusCountPerIp[ip] = make(map[int]int)
	}
	t.statusCountPerIp[ip][statusCode]++
}

func (t *IPTracker) GetHits(ip string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.hits[ip]
}

func getDiskUsage(path string) string {
	// Create a Statfs_t struct
	var stat unix.Statfs_t

	// Get file system stats for the given path
	err := unix.Statfs(path, &stat)
	if err != nil {
		return ""
	}

	// Calculate total, free, used, and available space in bytes
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	// Convert to megabytes
	totalMB := total / (1024 * 1024)
	usedMB := used / (1024 * 1024)
	availableMB := available / (1024 * 1024)
	freeMB := free / (1024 * 1024)

	// Calculate percentage of used space
	percentUsed := (float64(used) / float64(total)) * 100

	// Format the output
	return fmt.Sprintf(
		"Path: %s Total: %d MB Used: %d MB (%.2f%%) Available: %d MB Free: %d MB",
		path, totalMB, usedMB, percentUsed, availableMB, freeMB,
	)
}

func (t *IPTracker) GetTrackerInfo() map[string]interface{} {
	t.mu.Lock()
	defer t.mu.Unlock()

	var stats unix.Sysinfo_t

	err := unix.Sysinfo(&stats)
	if err != nil {
		fmt.Println("Error fetching system memory info:", err)
	}

	totalMemoryMB := uint64(stats.Totalram) * uint64(stats.Unit) / (1024 * 1024)
	freeMemoryMB := uint64(stats.Freeram) * uint64(stats.Unit) / (1024 * 1024)
	usedMemoryMB := totalMemoryMB - freeMemoryMB

	// Convert uptime (in seconds) to a human-readable format
	uptimeSeconds := int(stats.Uptime)
	days := uptimeSeconds / 86400
	hours := (uptimeSeconds % 86400) / 3600
	minutes := (uptimeSeconds % 3600) / 60
	seconds := uptimeSeconds % 60
	var uptime = fmt.Sprintf("%dd %02dh %02dm %02ds", days, hours, minutes, seconds)

	// Extract and scale the load averages
	load1 := float64(stats.Loads[0]) / 65536.0
	load5 := float64(stats.Loads[1]) / 65536.0
	load15 := float64(stats.Loads[2]) / 65536.0

	// Format the load averages
	var loadAverage = fmt.Sprintf("1-min: %.2f, 5-min: %.2f, 15-min: %.2f", load1, load5, load15)
	var usage = getDiskUsage("/")

	return map[string]interface{}{
		"hits":               t.hits,
		"banned":             t.banned,
		"statusCountPerIp":   t.statusCountPerIp,
		"system.memTotalMB":  totalMemoryMB,
		"system.memFreeMB":   freeMemoryMB,
		"system.memUsedMB":   usedMemoryMB,
		"system.uptime":      uptime,
		"system.loadAverage": loadAverage,
		"system.diskUsage":   usage,
	}
}

func (t *IPTracker) UnbanAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.banned = make(map[string]time.Time)
	log.Println("All IPs have been unbanned.")
}

func serve(backendURL *url.URL, disableBan bool, hit404threshold int, banDurantionInMinutes int, modifyHost bool) {
	defer wg.Done()

	// List all embedded files.
	fs.WalkDir(staticFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println("Embedded file:", path)
		return nil
	})

	tracker := NewIPTracker(hit404threshold, time.Duration(banDurantionInMinutes)*time.Minute) // Ban after x 404s, ban lasts 1 minute

	// these one where not bad, should perhaps be aligned
	// https://github.com/stevensouza/jamonapi/blob/4a5f2dd43fd276271c92b54f1c66eeb83366ad0a/jamon/src/main/java/com/jamonapi/RangeHolder.java#L53-L65
	buckets := []float64{
		0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8, 25.6, 51.12, 102.4, 204.8,
	}
	var perPathStats = NewPerPathStats(buckets)
	bucketStats := NewBucketStats(buckets)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip, _, _ = net.SplitHostPort(r.RemoteAddr) // Extract the IP without the port
		} else {
			// apparently X-Forwarded-For: <client>, <proxy1> we want to only keep the first value
			ip = strings.TrimSpace(strings.Split(ip, ",")[0])
		}

		start := time.Now()
		hits := tracker.GetHits(ip)

		log.Printf("Access log: method=%s url=%s ip=%s hits=%d", r.Method, r.URL.String(), ip, hits)

		if !disableBan && tracker.CheckBan(ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("Access log: method=%s url=%s ip=%s hits=%d (blocked)", r.Method, r.URL.String(), ip, hits)
			return
		}
		if modifyHost {
			r.Host = backendURL.Host

		}
		reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)
		reverseProxy.ModifyResponse = func(resp *http.Response) error {
			if resp.StatusCode == http.StatusNotFound {
				tracker.IncrementHit(ip)
			}
			tracker.IncrementStatus(ip, resp.StatusCode)

			hits := tracker.GetHits(ip)
			duration := time.Since(start).Seconds()

			stats := perPathStats.GetStatsForPath(CleanPath(r.URL.Path))
			stats.Record(duration)

			bucketStats.Record(duration)

			log.Printf("Access log: method=%s url=%s ip=%s hits=%d status=%v duration=%.3f", r.Method, r.URL.String(), ip, hits, resp.StatusCode, duration)
			return nil
		}
		reverseProxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/__banme/api/info", func(w http.ResponseWriter, r *http.Request) {
		info := tracker.GetTrackerInfo()
		info["percentiles.buckets"] = bucketStats.buckets
		info["percentiles.bucketCounts"] = bucketStats.bucketCounts
		info["percentiles.50"] = bucketStats.GetPercentile(50)
		info["percentiles.90"] = bucketStats.GetPercentile(90)
		info["percentiles.95"] = bucketStats.GetPercentile(95)
		info["percentiles.98"] = bucketStats.GetPercentile(98)
		info["percentiles.99"] = bucketStats.GetPercentile(99)

		info["percentiles.byPath"] = perPathStats.GetAllPercentiles()

		w.Header().Set("Content-Type", "application/json")
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			http.Error(w, "Failed to encode debug info", http.StatusInternalServerError)
			log.Printf("Failed to encode debug info: %v", err)
		}
		w.Write(jsonData)
	})

	http.HandleFunc("/__banme/api/unban", func(w http.ResponseWriter, r *http.Request) {
		tracker.UnbanAll()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("All IPs have been unbanned."))
	})

	subStaticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/__banme/", http.StripPrefix("/__banme/", http.FileServer(http.FS(subStaticFS))))

	log.Printf("Reverse proxy is running on :8000 for %s, hit404threshold=%v, banDurantionInMinutes=%v", backendURL, hit404threshold, banDurantionInMinutes)
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
