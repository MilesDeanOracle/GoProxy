package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
)

// Status describes the current proxy server runtime state.
type Status struct {
	Running     bool      `json:"running"`
	StartedAt   time.Time `json:"startedAt"`
	SOCKS5Addr  string    `json:"socks5Addr"`
	HTTPAddr    string    `json:"httpAddr"`
	ActiveConns int64     `json:"activeConns"`
	TotalConns  int64     `json:"totalConns"`
}

// Server manages proxy listeners and active connections.
type Server struct {
	cfg       config.Config
	collector *stats.Collector

	mu        sync.RWMutex
	running   bool
	startedAt time.Time
	cancel    context.CancelFunc
	listeners []net.Listener
	socksAddr string
	httpAddr  string

	sem chan struct{}

	connMu sync.Mutex
	conns  map[net.Conn]struct{}

	acceptWg sync.WaitGroup
	connWg   sync.WaitGroup
}

// NewServer creates a proxy server from validated runtime config.
func NewServer(cfg config.Config, collector *stats.Collector) *Server {
	return &Server{
		cfg:       cfg,
		collector: collector,
		sem:       make(chan struct{}, cfg.Relay.MaxConnections),
		conns:     make(map[net.Conn]struct{}),
	}
}

// Start opens all enabled listeners and begins accepting connections.
func (s *Server) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := config.Validate(s.cfg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("proxy server is already running")
	}

	serverCtx, cancel := context.WithCancel(ctx)
	listeners, err := s.openListeners()
	if err != nil {
		cancel()
		for _, listener := range listeners {
			_ = listener.Close()
		}
		s.socksAddr = ""
		s.httpAddr = ""
		return err
	}

	s.cancel = cancel
	s.listeners = listeners
	s.running = true
	s.startedAt = time.Now()

	for _, listener := range listeners {
		ln := listener
		protocol := listenerProtocol(ln)
		s.acceptWg.Add(1)
		go s.acceptLoop(serverCtx, protocol, ln)
	}

	go func() {
		<-serverCtx.Done()
		_ = s.Stop()
	}()

	return nil
}

// Stop closes listeners and active connections, then waits for goroutines to exit.
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}

	cancel := s.cancel
	listeners := append([]net.Listener(nil), s.listeners...)
	s.running = false
	s.cancel = nil
	s.listeners = nil
	s.socksAddr = ""
	s.httpAddr = ""
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, listener := range listeners {
		_ = listener.Close()
	}

	s.acceptWg.Wait()

	for _, conn := range s.activeConnections() {
		closeConn(conn)
	}
	s.connWg.Wait()

	return nil
}

// Status returns the current proxy server state and connection counters.
func (s *Server) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := s.collector.Snapshot()
	return Status{
		Running:     s.running,
		StartedAt:   s.startedAt,
		SOCKS5Addr:  s.socksAddr,
		HTTPAddr:    s.httpAddr,
		ActiveConns: snapshot.ActiveConns,
		TotalConns:  snapshot.TotalConns,
	}
}

// Stats returns the current collector snapshot.
func (s *Server) Stats() stats.Stats {
	return s.collector.Snapshot()
}

func (s *Server) openListeners() ([]net.Listener, error) {
	var listeners []net.Listener

	if s.cfg.Server.SOCKS5.Enabled {
		listener, err := net.Listen("tcp", networkAddress(s.cfg.Server.SOCKS5.Host, s.cfg.Server.SOCKS5.Port))
		if err != nil {
			return listeners, fmt.Errorf("listen socks5: %w", err)
		}
		listeners = append(listeners, &protocolListener{Listener: listener, protocol: "socks5"})
		s.socksAddr = listener.Addr().String()
	}

	if s.cfg.Server.HTTP.Enabled {
		listener, err := net.Listen("tcp", networkAddress(s.cfg.Server.HTTP.Host, s.cfg.Server.HTTP.Port))
		if err != nil {
			return listeners, fmt.Errorf("listen http: %w", err)
		}
		listeners = append(listeners, &protocolListener{Listener: listener, protocol: "http"})
		s.httpAddr = listener.Addr().String()
	}

	return listeners, nil
}

func (s *Server) acceptLoop(ctx context.Context, protocol string, listener net.Listener) {
	defer s.acceptWg.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}

		select {
		case s.sem <- struct{}{}:
		default:
			closeConn(conn)
			continue
		}

		s.registerConn(conn)
		s.connWg.Add(1)
		go func() {
			defer s.connWg.Done()
			defer func() { <-s.sem }()
			defer s.unregisterConn(conn)
			defer closeConn(conn)

			switch protocol {
			case "socks5":
				_ = s.handleSOCKS5(ctx, conn)
			case "http":
				_ = s.handleHTTPConnect(ctx, conn)
			}
		}()
	}
}

func (s *Server) registerConn(conn net.Conn) {
	s.connMu.Lock()
	s.conns[conn] = struct{}{}
	s.connMu.Unlock()
	s.collector.ConnOpened()
}

func (s *Server) unregisterConn(conn net.Conn) {
	s.connMu.Lock()
	delete(s.conns, conn)
	s.connMu.Unlock()
	s.collector.ConnClosed()
}

func (s *Server) activeConnections() []net.Conn {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	conns := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	return conns
}

type protocolListener struct {
	net.Listener
	protocol string
}

func listenerProtocol(listener net.Listener) string {
	if pl, ok := listener.(*protocolListener); ok {
		return pl.protocol
	}
	return ""
}
