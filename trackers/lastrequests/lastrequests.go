package lastrequests

import "time"

// RequestInfo represents a record with URL, status code, and user agent
type RequestInfo struct {
	FullURL    string    `json:"fullURL"`
	StatusCode int       `json:"statusCode"`
	UserAgent  string    `json:"userAgent"`
	StartTime  time.Time `json:"startTime"`
	Duration   float64   `json:"duration"`
	Ip         string    `json:"ip"`
}

// RingBuffer is a circular buffer to hold the last x RequestInfo records
type RingBuffer struct {
	buffer   []RequestInfo
	capacity int
	head     int
	size     int
}

// NewRingBuffer creates a new RingBuffer with the specified capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer:   make([]RequestInfo, capacity),
		capacity: capacity,
		head:     0,
		size:     0,
	}
}

// Add adds a new RequestInfo to the ring buffer
func (rb *RingBuffer) Add(record RequestInfo) {
	rb.buffer[rb.head] = record
	rb.head = (rb.head + 1) % rb.capacity
	if rb.size < rb.capacity {
		rb.size++
	}
}

// GetAll retrieves all the records in the buffer in order of insertion
func (rb *RingBuffer) GetAll() []RequestInfo {
	result := make([]RequestInfo, rb.size)
	for i := 0; i < rb.size; i++ {
		index := (rb.head - rb.size + i + rb.capacity) % rb.capacity
		result[i] = rb.buffer[index]
	}
	return result
}
