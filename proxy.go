package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
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

func (t *IPTracker) GetTrackerInfo() map[string]interface{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	return map[string]interface{}{
		"hits":             t.hits,
		"banned":           t.banned,
		"statusCountPerIp": t.statusCountPerIp,
	}
}

func (t *IPTracker) UnbanAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.banned = make(map[string]time.Time)
	log.Println("All IPs have been unbanned.")
}

func serve(backendURL *url.URL, disableBan bool) {
	defer wg.Done()

	tracker := NewIPTracker(50, 1*time.Minute) // Ban after x 404s, ban lasts 1 minute

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip, _, _ = net.SplitHostPort(r.RemoteAddr) // Extract the IP without the port
		}
		hits := tracker.GetHits(ip)

		log.Printf("Access log: method=%s url=%s ip=%s hits=%d", r.Method, r.URL.String(), ip, hits)

		if !disableBan && tracker.CheckBan(ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("Access log: method=%s url=%s ip=%s hits=%d (blocked)", r.Method, r.URL.String(), ip, hits)
			return
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)
		reverseProxy.ModifyResponse = func(resp *http.Response) error {
			if resp.StatusCode == http.StatusNotFound {
				tracker.IncrementHit(ip)
			}
			tracker.IncrementStatus(ip, resp.StatusCode)

			hits := tracker.GetHits(ip)
			log.Printf("Access log: method=%s url=%s ip=%s hits=%d status=%s", r.Method, r.URL.String(), ip, hits, resp.StatusCode)
			return nil
		}
		reverseProxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/__banme/api/info", func(w http.ResponseWriter, r *http.Request) {
		info := tracker.GetTrackerInfo()
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

	log.Printf("Reverse proxy is running on :8000 for %s", backendURL)
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
