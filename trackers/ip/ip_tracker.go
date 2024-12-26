package ip

import (
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

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
