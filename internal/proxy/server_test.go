package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
)

func TestSOCKS5ConnectIPv4(t *testing.T) {
	echoAddr := startEchoServer(t)
	targetHost, targetPort := splitHostPort(t, echoAddr)

	server := startTestServer(t, testConfig(freePort(t), 0, 8))
	conn := socks5Connect(t, server.Status().SOCKS5Addr, targetHost, targetPort, socksAddrIPv4)
	defer conn.Close()

	assertEcho(t, conn, "socks-ipv4")
}

func TestSOCKS5ConnectDomain(t *testing.T) {
	echoAddr := startEchoServer(t)
	_, targetPort := splitHostPort(t, echoAddr)

	server := startTestServer(t, testConfig(freePort(t), 0, 8))
	conn := socks5Connect(t, server.Status().SOCKS5Addr, "localhost", targetPort, socksAddrDomain)
	defer conn.Close()

	assertEcho(t, conn, "socks-domain")
}

func TestHTTPConnect(t *testing.T) {
	echoAddr := startEchoServer(t)
	server := startTestServer(t, testConfig(0, freePort(t), 8))

	conn, err := net.Dial("tcp", server.Status().HTTPAddr)
	if err != nil {
		t.Fatalf("dial http proxy: %v", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", echoAddr, echoAddr); err != nil {
		t.Fatalf("write connect request: %v", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read http connect status: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("expected 200 status, got %q", statusLine)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read http connect header: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	connections := server.ActiveConnections()
	if len(connections) != 1 {
		t.Fatalf("expected 1 active connection, got %d", len(connections))
	}
	if connections[0].Protocol != "http" || connections[0].TargetAddr != echoAddr {
		t.Fatalf("unexpected active connection snapshot: %+v", connections[0])
	}

	if _, err := conn.Write([]byte("http-connect")); err != nil {
		t.Fatalf("write tunnel data: %v", err)
	}
	readExact(t, reader, "http-connect")
}

func TestHTTPConnectWithoutHostHeader(t *testing.T) {
	echoAddr := startEchoServer(t)
	server := startTestServer(t, testConfig(0, freePort(t), 8))

	conn, err := net.Dial("tcp", server.Status().HTTPAddr)
	if err != nil {
		t.Fatalf("dial http proxy: %v", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\n\r\n", echoAddr); err != nil {
		t.Fatalf("write connect request: %v", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read http connect status: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("expected 200 status, got %q", statusLine)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read http connect header: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	if _, err := conn.Write([]byte("http-connect-no-host")); err != nil {
		t.Fatalf("write tunnel data: %v", err)
	}
	readExact(t, reader, "http-connect-no-host")
}

func TestHTTPForwardProxy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/through-proxy" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("forward-ok"))
	}))
	t.Cleanup(target.Close)

	server := startTestServer(t, testConfig(0, freePort(t), 8))
	conn, err := net.Dial("tcp", server.Status().HTTPAddr)
	if err != nil {
		t.Fatalf("dial http proxy: %v", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "GET %s/through-proxy HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target.URL, strings.TrimPrefix(target.URL, "http://")); err != nil {
		t.Fatalf("write forward request: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read forwarded response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 status, got %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read forwarded body: %v", err)
	}
	if string(body) != "forward-ok" {
		t.Fatalf("expected forward-ok, got %q", string(body))
	}
}

func TestServerStartStop(t *testing.T) {
	server := startTestServer(t, testConfig(freePort(t), freePort(t), 8))

	status := server.Status()
	if !status.Running {
		t.Fatal("expected server to be running")
	}
	if status.SOCKS5Addr == "" {
		t.Fatal("expected socks5 listener address")
	}
	if status.HTTPAddr == "" {
		t.Fatal("expected http listener address")
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("stop server: %v", err)
	}
	if server.Status().Running {
		t.Fatal("expected server to be stopped")
	}
}

func TestServerRejectsConnectionsOverLimit(t *testing.T) {
	server := startTestServer(t, testConfig(0, freePort(t), 1))

	first, err := net.Dial("tcp", server.Status().HTTPAddr)
	if err != nil {
		t.Fatalf("dial first connection: %v", err)
	}
	defer first.Close()

	waitFor(t, func() bool {
		return server.Stats().ActiveConns == 1
	})

	second, err := net.Dial("tcp", server.Status().HTTPAddr)
	if err != nil {
		t.Fatalf("dial second connection: %v", err)
	}
	defer second.Close()

	if err := second.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, err = second.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("expected second connection to be closed")
	}
}

func startTestServer(t *testing.T, cfg config.Config) *Server {
	t.Helper()

	collector := stats.NewCollector()
	server := NewServer(cfg, collector)
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Fatalf("stop server cleanup: %v", err)
		}
	})
	return server
}

func testConfig(socksPort, httpPort, maxConns int) config.Config {
	cfg := config.Default()
	cfg.Server.SOCKS5.Host = "127.0.0.1"
	cfg.Server.HTTP.Host = "127.0.0.1"
	cfg.Relay.DialTimeoutSec = 2
	cfg.Relay.ReadTimeoutSec = 2
	cfg.Relay.KeepAliveSec = 1
	cfg.Relay.MaxConnections = maxConns

	if socksPort > 0 {
		cfg.Server.SOCKS5.Enabled = true
		cfg.Server.SOCKS5.Port = socksPort
	} else {
		cfg.Server.SOCKS5.Enabled = false
		cfg.Server.SOCKS5.Port = 1080
	}

	if httpPort > 0 {
		cfg.Server.HTTP.Enabled = true
		cfg.Server.HTTP.Port = httpPort
	} else {
		cfg.Server.HTTP.Enabled = false
		cfg.Server.HTTP.Port = 8080
	}

	return cfg
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func startEchoServer(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()

	t.Cleanup(func() {
		close(done)
		_ = listener.Close()
	})

	return listener.Addr().String()
}

func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()

	host, portValue, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return host, port
}

func socks5Connect(t *testing.T, proxyAddr, targetHost string, targetPort int, atyp byte) net.Conn {
	t.Helper()

	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("dial socks proxy: %v", err)
	}

	if _, err := conn.Write([]byte{socksVersion5, 1, socksMethodNoAuth}); err != nil {
		t.Fatalf("write socks greeting: %v", err)
	}
	greeting := make([]byte, 2)
	if _, err := io.ReadFull(conn, greeting); err != nil {
		t.Fatalf("read socks greeting: %v", err)
	}
	if greeting[0] != socksVersion5 || greeting[1] != socksMethodNoAuth {
		t.Fatalf("unexpected socks greeting response: %v", greeting)
	}

	request := []byte{socksVersion5, socksCmdConnect, 0x00, atyp}
	switch atyp {
	case socksAddrIPv4:
		ip := net.ParseIP(targetHost).To4()
		if ip == nil {
			t.Fatalf("target host is not ipv4: %s", targetHost)
		}
		request = append(request, ip...)
	case socksAddrDomain:
		if len(targetHost) > 255 {
			t.Fatalf("target host too long: %s", targetHost)
		}
		request = append(request, byte(len(targetHost)))
		request = append(request, []byte(targetHost)...)
	default:
		t.Fatalf("unsupported test atyp: %d", atyp)
	}
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(targetPort))
	request = append(request, portBytes...)

	if _, err := conn.Write(request); err != nil {
		t.Fatalf("write socks request: %v", err)
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		t.Fatalf("read socks response header: %v", err)
	}
	if header[1] != socksReplySucceeded {
		t.Fatalf("expected socks success reply, got %d", header[1])
	}
	switch header[3] {
	case socksAddrIPv4:
		_, err = io.ReadFull(conn, make([]byte, net.IPv4len+2))
	case socksAddrIPv6:
		_, err = io.ReadFull(conn, make([]byte, net.IPv6len+2))
	default:
		t.Fatalf("unexpected socks response address type: %d", header[3])
	}
	if err != nil {
		t.Fatalf("read socks response address: %v", err)
	}

	return conn
}

func assertEcho(t *testing.T, conn net.Conn, message string) {
	t.Helper()

	if _, err := conn.Write([]byte(message)); err != nil {
		t.Fatalf("write echo message: %v", err)
	}
	readExact(t, conn, message)
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition did not become true")
}
