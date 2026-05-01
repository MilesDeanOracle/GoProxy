package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gitee.com/jiuhuidalan1/goproxy/internal/logger"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
	"go.uber.org/zap"
)

const serverStopGracePeriod = 3 * time.Second
const routeLogSource = "route"

// Status describes the current proxy server runtime state.
type Status struct {
	Running     bool   `json:"running"`
	StartedAt   string `json:"startedAt"`
	SOCKS5Addr  string `json:"socks5Addr"`
	HTTPAddr    string `json:"httpAddr"`
	ActiveConns int64  `json:"activeConns"`
	TotalConns  int64  `json:"totalConns"`
}

// ConnectionSnapshot describes an active proxied connection for the desktop UI.
type ConnectionSnapshot struct {
	ID            int64  `json:"id"`
	Protocol      string `json:"protocol"`
	ClientAddr    string `json:"clientAddr"`
	TargetAddr    string `json:"targetAddr"`
	RouteRuleName string `json:"routeRuleName"`
	OutboundIP    string `json:"outboundIp"`
	OutboundIface string `json:"outboundIface"`
	UploadBytes   int64  `json:"uploadBytes"`
	DownloadBytes int64  `json:"downloadBytes"`
	OpenedAt      string `json:"openedAt"`
}

type trackedConn struct {
	id            int64
	protocol      string
	clientAddr    string
	targetAddr    string
	routeRuleName string
	outboundIP    string
	outboundIface string
	uploadBytes   atomic.Int64
	downloadBytes atomic.Int64
	openedAt      time.Time
}

// Server manages proxy listeners and active connections.
type Server struct {
	cfg       config.Config
	collector *stats.Collector
	auth      *AuthManager
	logger    *logger.Manager
	routeMu   sync.RWMutex
	routeOn   bool
	route     RoutePolicyEngine

	mu        sync.RWMutex
	running   bool
	startedAt time.Time
	cancel    context.CancelFunc
	listeners []net.Listener
	socksAddr string
	httpAddr  string

	sem chan struct{}

	nextConnID int64
	connMu     sync.Mutex
	conns      map[net.Conn]*trackedConn

	trimMu    sync.Mutex
	trimTimer *time.Timer

	acceptWg sync.WaitGroup
	connWg   sync.WaitGroup

	stopGracePeriod time.Duration
}

// SetLogger updates the runtime logger used by route diagnostics.
func (s *Server) SetLogger(logManager *logger.Manager) {
	s.mu.Lock()
	s.logger = logManager
	s.mu.Unlock()
}

// NewServer creates a proxy server from validated runtime config.
func NewServer(cfg config.Config, collector *stats.Collector) *Server {
	mapSize := cfg.Relay.MaxConnections
	if mapSize > 10000 {
		mapSize = 10000
	}
	return &Server{
		cfg:       cfg,
		collector: collector,
		auth:      NewAuthManager(cfg.Auth),
		sem:       make(chan struct{}, cfg.Relay.MaxConnections),
		conns:     make(map[net.Conn]*trackedConn, mapSize),

		stopGracePeriod: serverStopGracePeriod,
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

// Stop closes listeners and active connections, then waits briefly for goroutines to exit.
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

	for _, conn := range s.activeConnections() {
		closeConn(conn)
	}

	gracePeriod := s.stopGracePeriod
	_ = waitGroupWithTimeout(&s.acceptWg, gracePeriod)
	if !waitGroupWithTimeout(&s.connWg, gracePeriod) {
		for _, conn := range s.activeConnections() {
			closeConn(conn)
		}
		_ = waitGroupWithTimeout(&s.connWg, 500*time.Millisecond)
	}
	s.cancelMemoryTrim()
	debug.FreeOSMemory()

	return nil
}

// Status returns the current proxy server state and connection counters.
func (s *Server) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := s.collector.Snapshot()
	var startedAt string
	if !s.startedAt.IsZero() {
		startedAt = s.startedAt.Format(time.RFC3339)
	}
	return Status{
		Running:     s.running,
		StartedAt:   startedAt,
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

// TickStats computes rate fields and returns the latest collector snapshot.
func (s *Server) TickStats() stats.Stats {
	return s.collector.Tick()
}

// SetAuthConfig updates authentication for newly processed requests.
func (s *Server) SetAuthConfig(cfg config.AuthConfig) {
	s.mu.Lock()
	s.cfg.Auth = cfg
	s.auth = NewAuthManager(cfg)
	s.mu.Unlock()
}

// SetRoutePolicy updates routing rules for newly established outbound connections.
func (s *Server) SetRoutePolicy(enabled bool, set config.RouteRuleSet) {
	var engine RoutePolicyEngine
	if enabled {
		engine = NewRoutePolicyEngine(set)
	}
	s.routeMu.Lock()
	s.routeOn = enabled
	s.route = engine
	s.routeMu.Unlock()
}

// ActiveConnections returns current active connection metadata.
func (s *Server) ActiveConnections() []ConnectionSnapshot {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	snapshots := make([]ConnectionSnapshot, 0, len(s.conns))
	for _, item := range s.conns {
		snapshots = append(snapshots, ConnectionSnapshot{
			ID:            item.id,
			Protocol:      item.protocol,
			ClientAddr:    item.clientAddr,
			TargetAddr:    item.targetAddr,
			RouteRuleName: item.routeRuleName,
			OutboundIP:    item.outboundIP,
			OutboundIface: item.outboundIface,
			UploadBytes:   item.uploadBytes.Load(),
			DownloadBytes: item.downloadBytes.Load(),
			OpenedAt:      item.openedAt.Format(time.RFC3339),
		})
	}
	return snapshots
}

func (s *Server) setConnRoute(conn net.Conn, decision RouteDecision) {
	s.connMu.Lock()
	if item := s.conns[conn]; item != nil {
		item.routeRuleName = decision.RuleName
		item.outboundIP = decision.LocalIP
		item.outboundIface = decision.InterfaceName
	}
	s.connMu.Unlock()
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

		s.registerConn(protocol, conn)
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

func (s *Server) registerConn(protocol string, conn net.Conn) {
	s.connMu.Lock()
	s.nextConnID++
	s.conns[conn] = &trackedConn{
		id:         s.nextConnID,
		protocol:   protocol,
		clientAddr: conn.RemoteAddr().String(),
		openedAt:   time.Now(),
	}
	s.connMu.Unlock()
	s.collector.ConnOpened()
}

func (s *Server) unregisterConn(conn net.Conn) {
	s.connMu.Lock()
	delete(s.conns, conn)
	remaining := len(s.conns)
	s.connMu.Unlock()
	s.collector.ConnClosed()
	if remaining == 0 {
		s.scheduleMemoryTrim()
	}
}

func (s *Server) setConnTarget(conn net.Conn, targetAddr string) {
	s.connMu.Lock()
	if item := s.conns[conn]; item != nil {
		item.targetAddr = targetAddr
	}
	s.connMu.Unlock()
}

func (s *Server) connByteCounters(conn net.Conn) (func(int64), func(int64)) {
	s.connMu.Lock()
	item := s.conns[conn]
	s.connMu.Unlock()

	return func(n int64) {
			s.addUpload(item, n)
		}, func(n int64) {
			s.addDownload(item, n)
		}
}

func (s *Server) addUpload(item *trackedConn, n int64) {
	if n <= 0 {
		return
	}
	s.collector.AddUpload(n)
	if item != nil {
		item.uploadBytes.Add(n)
	}
}

func (s *Server) addDownload(item *trackedConn, n int64) {
	if n <= 0 {
		return
	}
	s.collector.AddDownload(n)
	if item != nil {
		item.downloadBytes.Add(n)
	}
}

func (s *Server) authenticator() *AuthManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auth
}

func (s *Server) routeLogger() *logger.Manager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logger
}

func (s *Server) logRouteAccess(sourceAddr, targetAddr string, decision RouteDecision) {
	logManager := s.routeLogger()
	if logManager == nil || decision.RuleName == "" {
		return
	}
	logManager.Info(routeLogSource, formatRouteLogLine(sourceAddr, targetAddr, decision),
		zap.String("sourceIP", hostOnly(sourceAddr)),
		zap.String("accessIP", hostOnly(targetAddr)),
		zap.String("rule", decision.RuleName),
		zap.String("action", routeActionText(decision)),
	)
}

func (s *Server) logRouteFailure(sourceAddr, targetAddr string, decision RouteDecision, err error) {
	logManager := s.routeLogger()
	if logManager == nil || err == nil {
		return
	}
	if decision.RuleName == "" {
		return
	}
	message := formatRouteLogLine(sourceAddr, targetAddr, decision)
	fields := []zap.Field{
		zap.String("sourceIP", hostOnly(sourceAddr)),
		zap.String("accessIP", hostOnly(targetAddr)),
		zap.String("rule", decision.RuleName),
		zap.String("action", routeActionText(decision)),
		zap.String("mode", decision.OutboundMode),
	}
	if errors.Is(err, errRouteIntercepted) {
		logManager.Info(routeLogSource, message, fields...)
		return
	}
	fields = append(fields, zap.Error(err))
	logManager.Warn(routeLogSource, message, fields...)
}

func formatRouteLogLine(sourceAddr, targetAddr string, decision RouteDecision) string {
	return fmt.Sprintf("%s-%s-%s-触发规则(%s)-动作(%s)",
		time.Now().Format(time.RFC3339),
		hostOnly(sourceAddr),
		hostOnly(targetAddr),
		decision.RuleName,
		routeActionText(decision),
	)
}

func routeActionText(decision RouteDecision) string {
	switch decision.OutboundMode {
	case "intercept":
		return "拦截"
	case "interface":
		if decision.InterfaceName != "" {
			return "网卡(" + decision.InterfaceName + ")"
		}
		return "网卡"
	case "local_ip":
		if decision.LocalIP != "" {
			return "本地IP(" + decision.LocalIP + ")"
		}
		return "本地IP"
	default:
		return "直连"
	}
}

func hostOnly(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	return addr
}

func (s *Server) recordAuthFailure() {
	if s.collector != nil {
		s.collector.AuthFailed()
	}
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

func waitGroupWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	if timeout <= 0 {
		<-done
		return true
	}

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (s *Server) scheduleMemoryTrim() {
	s.trimMu.Lock()
	defer s.trimMu.Unlock()

	if s.trimTimer != nil {
		s.trimTimer.Stop()
	}
	s.trimTimer = time.AfterFunc(5*time.Second, func() {
		debug.FreeOSMemory()
		s.trimMu.Lock()
		s.trimTimer = nil
		s.trimMu.Unlock()
	})
}

func (s *Server) cancelMemoryTrim() {
	s.trimMu.Lock()
	defer s.trimMu.Unlock()

	if s.trimTimer != nil {
		s.trimTimer.Stop()
		s.trimTimer = nil
	}
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
