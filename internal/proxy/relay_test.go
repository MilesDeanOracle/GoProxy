package proxy

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
)

func TestRelayCopiesBothDirectionsAndCountsBytes(t *testing.T) {
	leftApp, leftRelay := tcpPair(t)
	rightRelay, rightApp := tcpPair(t)
	collector := stats.NewCollector()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- relay(ctx, leftRelay, rightRelay, time.Second, collector.AddUpload, collector.AddDownload)
	}()

	if _, err := leftApp.Write([]byte("hello")); err != nil {
		t.Fatalf("write upload: %v", err)
	}
	readExact(t, rightApp, "hello")

	if _, err := rightApp.Write([]byte("world")); err != nil {
		t.Fatalf("write download: %v", err)
	}
	readExact(t, leftApp, "world")

	_ = leftApp.Close()
	_ = rightApp.Close()
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("relay returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("relay did not stop")
	}

	snapshot := collector.Snapshot()
	if snapshot.UploadBytes != int64(len("hello")) {
		t.Fatalf("expected upload bytes %d, got %d", len("hello"), snapshot.UploadBytes)
	}
	if snapshot.DownloadBytes != int64(len("world")) {
		t.Fatalf("expected download bytes %d, got %d", len("world"), snapshot.DownloadBytes)
	}
}

func TestRelayDoesNotCloseIdleTunnelOnTimeout(t *testing.T) {
	leftApp, leftRelay := tcpPair(t)
	rightRelay, rightApp := tcpPair(t)
	collector := stats.NewCollector()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- relay(ctx, leftRelay, rightRelay, 100*time.Millisecond, collector.AddUpload, collector.AddDownload)
	}()

	time.Sleep(250 * time.Millisecond)

	select {
	case err := <-errCh:
		t.Fatalf("relay stopped while idle: %v", err)
	default:
	}

	if _, err := leftApp.Write([]byte("still-alive")); err != nil {
		t.Fatalf("write after idle period: %v", err)
	}
	readExact(t, rightApp, "still-alive")

	_ = leftApp.Close()
	_ = rightApp.Close()
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("relay returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("relay did not stop")
	}
}

func tcpPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp pair: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial tcp pair: %v", err)
	}

	select {
	case server := <-accepted:
		return client, server
	case <-time.After(2 * time.Second):
		t.Fatal("accept tcp pair timed out")
	}

	return nil, nil
}

func readExact(t *testing.T, reader io.Reader, want string) {
	t.Helper()

	buf := make([]byte, len(want))
	if _, err := io.ReadFull(reader, buf); err != nil {
		t.Fatalf("read %q: %v", want, err)
	}
	if string(buf) != want {
		t.Fatalf("expected %q, got %q", want, string(buf))
	}
}
