package proxy

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleHTTPConnect(ctx context.Context, conn net.Conn) error {
	timeout := time.Duration(s.cfg.Relay.ReadTimeoutSec) * time.Second
	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n"))
		return fmt.Errorf("read http connect request: %w", err)
	}
	defer req.Body.Close()

	if req.Method != http.MethodConnect {
		_, _ = conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\nConnection: close\r\n\r\n"))
		return fmt.Errorf("unsupported http proxy method %s", req.Method)
	}

	targetAddr := req.Host
	if targetAddr == "" {
		targetAddr = req.RequestURI
	}
	if !strings.Contains(targetAddr, ":") {
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n"))
		return fmt.Errorf("http connect target must include port: %s", targetAddr)
	}
	if _, _, err := net.SplitHostPort(targetAddr); err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n"))
		return fmt.Errorf("parse http connect target %s: %w", targetAddr, err)
	}

	dialer := &net.Dialer{
		Timeout:   time.Duration(s.cfg.Relay.DialTimeoutSec) * time.Second,
		KeepAlive: time.Duration(s.cfg.Relay.KeepAliveSec) * time.Second,
	}
	target, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\n\r\n"))
		return fmt.Errorf("dial http connect target %s: %w", targetAddr, err)
	}
	defer closeConn(target)

	setTCPKeepAlive(conn, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)
	setTCPKeepAlive(target, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)

	if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return err
	}

	clearDeadlines(conn, target)
	return relay(ctx, &bufferedConn{Conn: conn, reader: reader}, target, s.collector, timeout)
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	if c.reader != nil && c.reader.Buffered() > 0 {
		return c.reader.Read(p)
	}
	return c.Conn.Read(p)
}

func (c *bufferedConn) CloseWrite() error {
	type closeWriter interface {
		CloseWrite() error
	}

	if cw, ok := c.Conn.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return c.Conn.Close()
}
