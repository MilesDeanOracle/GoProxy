package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	relayBufferSize    = 32 * 1024
	statsFlushBytes    = 256 * 1024
	statsFlushInterval = time.Second
)

var relayBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, relayBufferSize)
	},
}

func relay(ctx context.Context, client, target net.Conn, writeTimeout time.Duration, onUpload, onDownload func(int64)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var firstErr error
	var ctxErr error
	var closeOnce sync.Once

	closeBoth := func() {
		closeOnce.Do(func() {
			closeConn(client)
			closeConn(target)
		})
	}

	errCh := make(chan error, 2)
	go func() {
		err := copyConn(target, client, writeTimeout, onUpload)
		closeWrite(target)
		errCh <- err
	}()
	go func() {
		err := copyConn(client, target, writeTimeout, onDownload)
		closeWrite(client)
		errCh <- err
	}()

	completed := 0
	for completed < 2 {
		select {
		case err := <-errCh:
			completed++
			if err != nil && !isExpectedCloseError(err) && firstErr == nil {
				firstErr = err
				cancel()
				closeBoth()
			}
		case <-ctx.Done():
			if ctxErr == nil {
				ctxErr = ctx.Err()
				closeBoth()
			}
		}
	}

	if firstErr != nil {
		return firstErr
	}
	if ctxErr != nil && !errors.Is(ctxErr, context.Canceled) {
		return ctxErr
	}
	return nil
}

func copyConn(dst, src net.Conn, writeTimeout time.Duration, onBytes func(int64)) error {
	buf := relayBufferPool.Get().([]byte)
	defer relayBufferPool.Put(buf)

	var pendingBytes int64
	lastFlush := time.Now()
	flushBytes := func(force bool) {
		if onBytes == nil || pendingBytes <= 0 {
			return
		}
		if !force && pendingBytes < statsFlushBytes && time.Since(lastFlush) < statsFlushInterval {
			return
		}
		onBytes(pendingBytes)
		pendingBytes = 0
		lastFlush = time.Now()
	}
	defer flushBytes(true)

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if writeTimeout > 0 {
				_ = dst.SetWriteDeadline(time.Now().Add(writeTimeout))
			}

			written, writeErr := dst.Write(buf[:n])
			if written > 0 && onBytes != nil {
				pendingBytes += int64(written)
				flushBytes(false)
			}
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

func closeWrite(conn net.Conn) {
	type closeWriter interface {
		CloseWrite() error
	}

	if cw, ok := conn.(closeWriter); ok {
		_ = cw.CloseWrite()
		return
	}
	_ = conn.Close()
}

func closeConn(conn net.Conn) {
	if conn != nil {
		_ = conn.Close()
	}
}

func setTCPKeepAlive(conn net.Conn, period time.Duration) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	if period > 0 {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(period)
	}
}

func clearDeadlines(conns ...net.Conn) {
	for _, conn := range conns {
		if conn != nil {
			_ = conn.SetDeadline(time.Time{})
		}
	}
}

func isExpectedCloseError(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "An existing connection was forcibly closed") ||
		strings.Contains(msg, "wsasend") ||
		strings.Contains(msg, "wsarecv")
}

func networkAddress(host string, port int) string {
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}
