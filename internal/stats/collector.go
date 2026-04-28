package stats

import "sync/atomic"

// Stats is a point-in-time snapshot of proxy runtime counters.
type Stats struct {
	ActiveConns   int64 `json:"activeConns"`
	TotalConns    int64 `json:"totalConns"`
	UploadBytes   int64 `json:"uploadBytes"`
	DownloadBytes int64 `json:"downloadBytes"`
}

// Collector tracks connection and byte counters using atomic operations.
type Collector struct {
	activeConns   atomic.Int64
	totalConns    atomic.Int64
	uploadBytes   atomic.Int64
	downloadBytes atomic.Int64
}

// NewCollector creates an empty stats collector.
func NewCollector() *Collector {
	return &Collector{}
}

// ConnOpened records a newly accepted proxied connection.
func (c *Collector) ConnOpened() {
	if c == nil {
		return
	}
	c.activeConns.Add(1)
	c.totalConns.Add(1)
}

// ConnClosed records a closed proxied connection.
func (c *Collector) ConnClosed() {
	if c == nil {
		return
	}
	c.activeConns.Add(-1)
}

// AddUpload records bytes sent from the client to the target.
func (c *Collector) AddUpload(n int64) {
	if c == nil || n <= 0 {
		return
	}
	c.uploadBytes.Add(n)
}

// AddDownload records bytes sent from the target to the client.
func (c *Collector) AddDownload(n int64) {
	if c == nil || n <= 0 {
		return
	}
	c.downloadBytes.Add(n)
}

// Snapshot returns the current counter values.
func (c *Collector) Snapshot() Stats {
	if c == nil {
		return Stats{}
	}
	return Stats{
		ActiveConns:   c.activeConns.Load(),
		TotalConns:    c.totalConns.Load(),
		UploadBytes:   c.uploadBytes.Load(),
		DownloadBytes: c.downloadBytes.Load(),
	}
}
