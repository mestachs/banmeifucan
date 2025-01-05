package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"reverseproxy/trackers/active"
	"reverseproxy/trackers/buckets"
	"reverseproxy/trackers/ip"
)

// Embed the entire "static" folder.
//
//go:embed static/*
var staticFiles embed.FS

func serve(backendURL *url.URL, disableBan bool, hit404threshold int, banDurantionInMinutes int, modifyHost bool) {
	defer wg.Done()

	tracker := ip.NewIPTracker(hit404threshold, time.Duration(banDurantionInMinutes)*time.Minute) // Ban after x 404s, ban lasts 1 minute

	// these one where not bad, should perhaps be aligned
	// https://github.com/stevensouza/jamonapi/blob/4a5f2dd43fd276271c92b54f1c66eeb83366ad0a/jamon/src/main/java/com/jamonapi/RangeHolder.java#L53-L65
	bucketsDef := []float64{
		0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8, 25.6, 51.12, 102.4, 204.8,
	}
	var perPathStats = buckets.NewPerPathStats(bucketsDef)
	bucketStats := buckets.NewBucketStats(bucketsDef)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		client_ip := r.Header.Get("X-Forwarded-For")
		if client_ip == "" {
			client_ip, _, _ = net.SplitHostPort(r.RemoteAddr) // Extract the IP without the port
		} else {
			// apparently X-Forwarded-For: <client>, <reverse_proxy>
			client_ip = strings.TrimSpace(strings.Split(client_ip, ",")[0])
		}

		start := time.Now()
		hits := tracker.GetHits(client_ip)

		log.Printf("Access log: method=%s url=%s ip=%s hits=%d", r.Method, r.URL.String(), client_ip, hits)

		if !disableBan && tracker.CheckBan(client_ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("Access log: method=%s url=%s ip=%s hits=%d (blocked)", r.Method, r.URL.String(), client_ip, hits)
			return
		}
		if modifyHost {
			r.Host = backendURL.Host

		}

		cleanedPath := CleanPath(r.URL.Path)

		connStats := active.RecordActiveConnection(cleanedPath)

		reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)
		reverseProxy.ModifyResponse = func(resp *http.Response) error {
			if resp.StatusCode == http.StatusNotFound {
				tracker.IncrementHit(client_ip)
			}
			tracker.IncrementStatus(client_ip, resp.StatusCode)

			hits := tracker.GetHits(client_ip)
			duration := time.Since(start).Seconds()

			stats := perPathStats.GetStatsForPath(cleanedPath)
			stats.Record(duration, resp.StatusCode)

			bucketStats.Record(duration, resp.StatusCode)

			log.Printf("Access log: method=%s url=%s ip=%s hits=%d status=%v duration=%.3f", r.Method, r.URL.String(), client_ip, hits, resp.StatusCode, duration)

			return nil
		}
		defer func() {
			connStats.StopActiveConnection()
		}()
		reverseProxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/__banme/api/info", func(w http.ResponseWriter, r *http.Request) {
		info := tracker.GetTrackerInfo()
		info["percentiles.buckets"] = bucketStats.Buckets()
		info["percentiles.bucketCounts"] = bucketStats.BucketCounts()
		info["percentiles.50"] = bucketStats.GetPercentile(50)
		info["percentiles.90"] = bucketStats.GetPercentile(90)
		info["percentiles.95"] = bucketStats.GetPercentile(95)
		info["percentiles.98"] = bucketStats.GetPercentile(98)
		info["percentiles.99"] = bucketStats.GetPercentile(99)
		info["percentiles.totalTime"] = bucketStats.TotalTime()
		info["percentiles.totalCount"] = bucketStats.TotalCount()

		percentilesByPath := perPathStats.GetAllPercentiles()

		info["percentiles.byPath"] = percentilesByPath

		for path, stats := range percentilesByPath {

			connnStats := active.GetActiveConnections(path)
			stats["active"] = connnStats.GetActiveConnections()
			stats["maxActive"] = connnStats.GetMaxActiveConnections()
		}

		info["percentiles.statusCount"] = bucketStats.StatusesCount

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
