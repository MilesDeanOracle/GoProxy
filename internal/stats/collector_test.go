package stats

import "testing"

func TestCollectorSnapshot(t *testing.T) {
	collector := NewCollector()

	collector.ConnOpened()
	collector.ConnOpened()
	collector.AddUpload(12)
	collector.AddDownload(34)
	collector.ConnClosed()

	snapshot := collector.Snapshot()
	if snapshot.ActiveConns != 1 {
		t.Fatalf("expected active connections 1, got %d", snapshot.ActiveConns)
	}
	if snapshot.TotalConns != 2 {
		t.Fatalf("expected total connections 2, got %d", snapshot.TotalConns)
	}
	if snapshot.UploadBytes != 12 {
		t.Fatalf("expected upload bytes 12, got %d", snapshot.UploadBytes)
	}
	if snapshot.DownloadBytes != 34 {
		t.Fatalf("expected download bytes 34, got %d", snapshot.DownloadBytes)
	}
}
