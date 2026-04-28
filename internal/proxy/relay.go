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

func relay(ctx context.Context, client, target net.Conn, writeTimeout time.Duration, onUpload, onDownload func(int64)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	recordErr := func(err error) {
		if err == nil || isExpectedCloseError(err) {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			cancel()
			closeConn(client)
			closeConn(target)
		})
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		err := copyConn(target, client, writeTimeout, onUpload)
		closeWrite(target)
		recordErr(err)
	}()
	go func() {
		defer wg.Done()
		err := copyConn(client, target, writeTimeout, onDownload)
		closeWrite(client)
		recordErr(err)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		closeConn(client)
		closeConn(target)
		<-done
		if errors.Is(ctx.Err(), context.Canceled) && firstErr == nil {
			return nil
		}
		if firstErr != nil {
			return firstErr
		}
		return ctx.Err()
	case <-done:
		return firstErr
	}
}

func copyConn(dst, src net.Conn, writeTimeout time.Duration, onBytes func(int64)) error {
	buf := make([]byte, 32*1024)

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if writeTimeout > 0 {
				_ = dst.SetWriteDeadline(time.Now().Add(writeTimeout))
			}

			written, writeErr := dst.Write(buf[:n])
			if written > 0 && onBytes != nil {
				onBytes(int64(written))
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
	if !ok || period <= 0 {
		return
	}
	_ = tcpConn.SetKeepAlive(true)
	_ = tcpConn.SetKeepAlivePeriod(period)
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
