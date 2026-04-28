package proxy

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
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

	if req.Method == http.MethodConnect {
		return s.handleHTTPTunnel(ctx, conn, reader, req, timeout)
	}

	return s.handleHTTPForward(ctx, conn, reader, req, timeout)
}

func (s *Server) handleHTTPTunnel(ctx context.Context, conn net.Conn, reader *bufio.Reader, req *http.Request, timeout time.Duration) error {
	targetAddr, err := connectTarget(req)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return err
	}
	s.setConnTarget(conn, targetAddr)

	target, err := s.dialProxyTarget(ctx, targetAddr)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return fmt.Errorf("dial http connect target %s: %w", targetAddr, err)
	}
	defer closeConn(target)

	setTCPKeepAlive(conn, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)
	setTCPKeepAlive(target, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)

	if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\nProxy-Agent: ProxyServer\r\n\r\n")); err != nil {
		return err
	}

	clearDeadlines(conn, target)
	return relay(ctx, &bufferedConn{Conn: conn, reader: reader}, target, timeout, func(n int64) {
		s.addConnUpload(conn, n)
	}, func(n int64) {
		s.addConnDownload(conn, n)
	})
}

func (s *Server) handleHTTPForward(ctx context.Context, conn net.Conn, reader *bufio.Reader, req *http.Request, timeout time.Duration) error {
	targetAddr, err := forwardTarget(req)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return err
	}
	s.setConnTarget(conn, targetAddr)

	target, err := s.dialProxyTarget(ctx, targetAddr)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return fmt.Errorf("dial http forward target %s: %w", targetAddr, err)
	}
	defer closeConn(target)

	setTCPKeepAlive(conn, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)
	setTCPKeepAlive(target, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)
	clearDeadlines(conn, target)

	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authorization")
	if req.URL != nil {
		req.URL.Scheme = ""
		req.URL.Host = ""
	}
	if err := req.Write(target); err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return fmt.Errorf("write forwarded http request: %w", err)
	}

	return relay(ctx, &bufferedConn{Conn: conn, reader: reader}, target, timeout, func(n int64) {
		s.addConnUpload(conn, n)
	}, func(n int64) {
		s.addConnDownload(conn, n)
	})
}

func (s *Server) dialProxyTarget(ctx context.Context, targetAddr string) (net.Conn, error) {
	timeout := time.Duration(s.cfg.Relay.DialTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: time.Duration(s.cfg.Relay.KeepAliveSec) * time.Second,
	}

	if ip := net.ParseIP(host); ip != nil {
		return dialer.DialContext(ctx, "tcp", targetAddr)
	}

	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
	if err != nil || len(addrs) == 0 {
		return dialer.DialContext(resolveCtx, "tcp", targetAddr)
	}

	ordered := make([]net.IPAddr, 0, len(addrs))
	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			ordered = append(ordered, addr)
		}
	}
	for _, addr := range addrs {
		if addr.IP.To4() == nil {
			ordered = append(ordered, addr)
		}
	}

	var lastErr error
	perAttempt := timeout
	if len(ordered) > 1 && timeout > 3*time.Second {
		perAttempt = 3 * time.Second
	}

	for _, addr := range ordered {
		select {
		case <-resolveCtx.Done():
			return nil, resolveCtx.Err()
		default:
		}

		attemptCtx, attemptCancel := context.WithTimeout(resolveCtx, perAttempt)
		candidate := net.JoinHostPort(addr.IP.String(), port)
		conn, err := dialer.DialContext(attemptCtx, "tcp", candidate)
		attemptCancel()
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

func connectTarget(req *http.Request) (string, error) {
	target := req.Host
	if target == "" {
		target = req.RequestURI
	}
	if target == "" && req.URL != nil {
		target = req.URL.Host
	}
	return normalizeProxyTarget(target, "443")
}

func forwardTarget(req *http.Request) (string, error) {
	target := ""
	defaultPort := "80"
	if req.URL != nil {
		target = req.URL.Host
		if req.URL.Scheme == "https" {
			defaultPort = "443"
		}
	}
	if target == "" {
		target = req.Host
	}
	return normalizeProxyTarget(target, defaultPort)
}

func normalizeProxyTarget(target, defaultPort string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" || target == "*" {
		return "", fmt.Errorf("proxy target is empty")
	}

	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)
		if err != nil {
			return "", fmt.Errorf("parse proxy target %s: %w", target, err)
		}
		target = parsed.Host
	}

	if host, port, err := net.SplitHostPort(target); err == nil {
		return net.JoinHostPort(host, port), nil
	}

	if strings.HasPrefix(target, "[") && strings.HasSuffix(target, "]") {
		target = strings.TrimPrefix(strings.TrimSuffix(target, "]"), "[")
	}

	if ip := net.ParseIP(target); ip != nil {
		return net.JoinHostPort(ip.String(), defaultPort), nil
	}

	if strings.Count(target, ":") > 1 {
		return net.JoinHostPort(target, defaultPort), nil
	}

	host := target
	if i := strings.LastIndex(target, ":"); i >= 0 {
		host = target[:i]
		port := target[i+1:]
		if host != "" && port != "" {
			return net.JoinHostPort(host, port), nil
		}
	}

	return net.JoinHostPort(host, defaultPort), nil
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
