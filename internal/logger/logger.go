package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const defaultRingSize = 1000

// Entry is the log payload sent to the desktop UI.
type Entry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

// Manager owns structured logging, file rotation, and the UI log ring buffer.
type Manager struct {
	zapLogger *zap.Logger
	ring      *RingBuffer
	closers   []io.Closer

	mu          sync.RWMutex
	subscribers []func(Entry)
}

// NewManager creates a log manager from runtime config.
func NewManager(cfg config.LogConfig, logPath string) (*Manager, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	writeSyncer, closers, err := buildWriteSyncer(cfg, logPath)
	if err != nil {
		return nil, err
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "message"

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		level,
	)

	return &Manager{
		zapLogger: zap.New(core),
		ring:      NewRingBuffer(defaultRingSize),
		closers:   closers,
	}, nil
}

// Subscribe registers a callback for new UI log entries.
func (m *Manager) Subscribe(fn func(Entry)) {
	if m == nil || fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, fn)
}

// Recent returns recent UI log entries.
func (m *Manager) Recent(n int) []Entry {
	if m == nil {
		return nil
	}
	return m.ring.Recent(n)
}

// Debug writes a debug-level entry.
func (m *Manager) Debug(source, message string, fields ...zap.Field) {
	m.log("debug", source, message, fields...)
}

// Info writes an info-level entry.
func (m *Manager) Info(source, message string, fields ...zap.Field) {
	m.log("info", source, message, fields...)
}

// Warn writes a warn-level entry.
func (m *Manager) Warn(source, message string, fields ...zap.Field) {
	m.log("warn", source, message, fields...)
}

// Error writes an error-level entry.
func (m *Manager) Error(source, message string, fields ...zap.Field) {
	m.log("error", source, message, fields...)
}

// Close flushes buffered logger state.
func (m *Manager) Close() error {
	if m == nil || m.zapLogger == nil {
		return nil
	}
	var syncErr error
	if err := m.zapLogger.Sync(); err != nil && !isIgnorableSyncError(err) {
		syncErr = err
	}
	for _, closer := range m.closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return syncErr
}

func (m *Manager) log(level, source, message string, fields ...zap.Field) {
	if m == nil {
		return
	}

	entry := Entry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   strings.ToUpper(level),
		Message: message,
		Source:  source,
	}

	m.ring.Add(entry)
	m.notify(entry)

	fields = append(fields, zap.String("source", source))
	switch level {
	case "debug":
		m.zapLogger.Debug(message, fields...)
	case "warn":
		m.zapLogger.Warn(message, fields...)
	case "error":
		m.zapLogger.Error(message, fields...)
	default:
		m.zapLogger.Info(message, fields...)
	}
}

func (m *Manager) notify(entry Entry) {
	m.mu.RLock()
	subscribers := append([]func(Entry){}, m.subscribers...)
	m.mu.RUnlock()

	for _, subscriber := range subscribers {
		subscriber(entry)
	}
}

func parseLevel(level string) (zapcore.LevelEnabler, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return nil, fmt.Errorf("unsupported log level %q", level)
	}
}

func buildWriteSyncer(cfg config.LogConfig, logPath string) (zapcore.WriteSyncer, []io.Closer, error) {
	var writers []zapcore.WriteSyncer
	var closers []io.Closer

	if cfg.Output == "console" || cfg.Output == "both" {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}
	if cfg.Output == "file" || cfg.Output == "both" {
		if logPath == "" {
			return nil, nil, fmt.Errorf("log path is required when file output is enabled")
		}
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create log directory: %w", err)
		}
		fileWriter := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			Compress:   false,
		}
		writers = append(writers, zapcore.AddSync(fileWriter))
		closers = append(closers, fileWriter)
	}
	if len(writers) == 0 {
		return nil, nil, fmt.Errorf("at least one log output is required")
	}
	return zapcore.NewMultiWriteSyncer(writers...), closers, nil
}

func isIgnorableSyncError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid argument") ||
		strings.Contains(msg, "inappropriate ioctl") ||
		strings.Contains(msg, "handle is invalid") ||
		strings.Contains(msg, "bad file descriptor")
}
