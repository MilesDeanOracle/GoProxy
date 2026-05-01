package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
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

	if !s.authenticateHTTPProxy(conn, req) {
		return errors.New("http proxy authentication failed")
	}

	if req.Method == http.MethodConnect {
		return s.handleHTTPTunnel(ctx, conn, reader, req, timeout)
	}

	return s.handleHTTPForward(ctx, conn, reader, req, timeout)
}

func (s *Server) authenticateHTTPProxy(conn net.Conn, req *http.Request) bool {
	auth := s.authenticator()
	if !auth.Enabled() {
		return true
	}

	username, password, ok := parseProxyBasicAuth(req.Header.Get("Proxy-Authorization"))
	if ok && auth.Validate(username, password) {
		return true
	}

	s.recordAuthFailure()
	_, _ = conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"GoProxy\"\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
	return false
}

func parseProxyBasicAuth(header string) (string, string, bool) {
	const prefix = "Basic "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return "", "", false
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

func (s *Server) handleHTTPTunnel(ctx context.Context, conn net.Conn, reader *bufio.Reader, req *http.Request, timeout time.Duration) error {
	targetAddr, err := connectTarget(req)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return err
	}
	s.setConnTarget(conn, targetAddr)

	target, decision, err := s.dialProxyTarget(ctx, "http", conn.RemoteAddr().String(), targetAddr)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return fmt.Errorf("dial http connect target %s: %w", targetAddr, err)
	}
	defer closeConn(target)
	s.setConnRoute(conn, decision)

	setTCPKeepAlive(conn, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)
	setTCPKeepAlive(target, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)

	if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\nProxy-Agent: GoProxy\r\n\r\n")); err != nil {
		return err
	}

	clearDeadlines(conn, target)
	onUpload, onDownload := s.connByteCounters(conn)
	return relay(ctx, &bufferedConn{Conn: conn, reader: reader}, target, timeout, onUpload, onDownload)
}

func (s *Server) handleHTTPForward(ctx context.Context, conn net.Conn, reader *bufio.Reader, req *http.Request, timeout time.Duration) error {
	targetAddr, err := forwardTarget(req)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return err
	}
	s.setConnTarget(conn, targetAddr)

	target, decision, err := s.dialProxyTarget(ctx, "http", conn.RemoteAddr().String(), targetAddr)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Length: 0\r\n\r\n"))
		return fmt.Errorf("dial http forward target %s: %w", targetAddr, err)
	}
	defer closeConn(target)
	s.setConnRoute(conn, decision)

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

	onUpload, onDownload := s.connByteCounters(conn)
	return relay(ctx, &bufferedConn{Conn: conn, reader: reader}, target, timeout, onUpload, onDownload)
}

func (s *Server) dialProxyTarget(ctx context.Context, protocol, sourceAddr, targetAddr string) (net.Conn, RouteDecision, error) {
	timeout := time.Duration(s.cfg.Relay.DialTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, RouteDecision{}, err
	}

	decision := s.routeDecision(RouteContext{
		Protocol:   protocol,
		TargetHost: host,
		TargetPort: port,
		IsIP:       net.ParseIP(host) != nil,
	})
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: time.Duration(s.cfg.Relay.KeepAliveSec) * time.Second,
	}
	if err := applyOutboundBinding(dialer, &decision); err != nil {
		s.logRouteFailure(sourceAddr, targetAddr, decision, err)
		return nil, decision, err
	}

	if ip := net.ParseIP(host); ip != nil {
		conn, err := dialer.DialContext(ctx, "tcp", targetAddr)
		if err == nil {
			fillOutboundLocalIP(conn, &decision)
			if decision.RuleName != "" {
				s.logRouteAccess(sourceAddr, conn.RemoteAddr().String(), decision)
			}
		}
		return conn, decision, err
	}

	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
	if err != nil || len(addrs) == 0 {
		conn, err := dialer.DialContext(resolveCtx, "tcp", targetAddr)
		if err == nil {
			fillOutboundLocalIP(conn, &decision)
			if decision.RuleName != "" {
				s.logRouteAccess(sourceAddr, conn.RemoteAddr().String(), decision)
			}
		}
		return conn, decision, err
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
			return nil, decision, resolveCtx.Err()
		default:
		}

		attemptCtx, attemptCancel := context.WithTimeout(resolveCtx, perAttempt)
		candidate := net.JoinHostPort(addr.IP.String(), port)
		conn, err := dialer.DialContext(attemptCtx, "tcp", candidate)
		attemptCancel()
		if err == nil {
			fillOutboundLocalIP(conn, &decision)
			if decision.RuleName != "" {
				s.logRouteAccess(sourceAddr, conn.RemoteAddr().String(), decision)
			}
			return conn, decision, nil
		}
		lastErr = err
	}

	return nil, decision, lastErr
}

func (s *Server) routeDecision(ctx RouteContext) RouteDecision {
	s.routeMu.RLock()
	enabled := s.routeOn
	engine := s.route
	s.routeMu.RUnlock()
	if !enabled || engine == nil {
		return RouteDecision{OutboundMode: "default"}
	}
	return engine.Match(ctx)
}

func fillOutboundLocalIP(conn net.Conn, decision *RouteDecision) {
	if conn == nil || decision == nil {
		return
	}
	if decision.LocalIP != "" {
		return
	}
	if tcpAddr, ok := conn.LocalAddr().(*net.TCPAddr); ok && tcpAddr != nil && tcpAddr.IP != nil {
		decision.LocalIP = tcpAddr.IP.String()
	}
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
