package active

import (
	"sync"
	"sync/atomic"
)

type ConnectionStats struct {
	ActiveConnections int64
	MaxConnections    int64
}

var activeConnections sync.Map

func GetActiveConnections(path string) *ConnectionStats {

	stats, _ := activeConnections.LoadOrStore(path, &ConnectionStats{})
	connStats := stats.(*ConnectionStats)
	return connStats
}

func RecordActiveConnection(cleanedPath string) *ConnectionStats {

	stats, _ := activeConnections.LoadOrStore(cleanedPath, &ConnectionStats{})
	connStats := stats.(*ConnectionStats)

	// Increment the active connections for the path
	atomic.AddInt64(&connStats.ActiveConnections, 1)

	// Update the max active connections if necessary
	for {
		currentMax := atomic.LoadInt64(&connStats.MaxConnections)
		if connStats.ActiveConnections > currentMax {
			// Attempt to update max active connections atomically
			if atomic.CompareAndSwapInt64(&connStats.MaxConnections, currentMax, connStats.ActiveConnections) {
				break
			}
		} else {
			break
		}
	}

	return connStats
}

func (connStats *ConnectionStats) StopActiveConnection() {
	atomic.AddInt64(&connStats.ActiveConnections, -1)
}

func (connStats *ConnectionStats) GetActiveConnections() int64 {
	return atomic.LoadInt64(&connStats.ActiveConnections)
}

func (connStats *ConnectionStats) GetMaxActiveConnections() int64 {
	return atomic.LoadInt64(&connStats.MaxConnections)
}
