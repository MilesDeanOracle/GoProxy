package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	socksVersion5 = 0x05

	socksMethodNoAuth       = 0x00
	socksMethodNoAcceptable = 0xff

	socksCmdConnect = 0x01

	socksAddrIPv4   = 0x01
	socksAddrDomain = 0x03
	socksAddrIPv6   = 0x04

	socksReplySucceeded          = 0x00
	socksReplyGeneralFailure     = 0x01
	socksReplyNetworkUnreachable = 0x03
	socksReplyHostUnreachable    = 0x04
	socksReplyCommandUnsupported = 0x07
	socksReplyAddrUnsupported    = 0x08
)

func (s *Server) handleSOCKS5(ctx context.Context, conn net.Conn) error {
	timeout := time.Duration(s.cfg.Relay.ReadTimeoutSec) * time.Second
	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	if err := negotiateSOCKS5(conn); err != nil {
		return err
	}

	targetAddr, err := readSOCKS5ConnectRequest(conn)
	if err != nil {
		var requestErr *socksRequestError
		if errors.As(err, &requestErr) {
			_ = writeSOCKS5Reply(conn, requestErr.reply, nil)
		} else {
			_ = writeSOCKS5Reply(conn, socksReplyGeneralFailure, nil)
		}
		return err
	}
	s.setConnTarget(conn, targetAddr)

	dialer := &net.Dialer{
		Timeout:   time.Duration(s.cfg.Relay.DialTimeoutSec) * time.Second,
		KeepAlive: time.Duration(s.cfg.Relay.KeepAliveSec) * time.Second,
	}
	target, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		_ = writeSOCKS5Reply(conn, socksReplyHostUnreachable, nil)
		return fmt.Errorf("dial socks5 target %s: %w", targetAddr, err)
	}
	defer closeConn(target)

	setTCPKeepAlive(conn, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)
	setTCPKeepAlive(target, time.Duration(s.cfg.Relay.KeepAliveSec)*time.Second)

	if err := writeSOCKS5Reply(conn, socksReplySucceeded, target.LocalAddr()); err != nil {
		return err
	}

	clearDeadlines(conn, target)
	onUpload, onDownload := s.connByteCounters(conn)
	return relay(ctx, conn, target, timeout, onUpload, onDownload)
}

func negotiateSOCKS5(conn net.Conn) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return fmt.Errorf("read socks5 greeting: %w", err)
	}
	if header[0] != socksVersion5 {
		return fmt.Errorf("unsupported socks version %d", header[0])
	}

	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return fmt.Errorf("read socks5 methods: %w", err)
	}

	for _, method := range methods {
		if method == socksMethodNoAuth {
			_, err := conn.Write([]byte{socksVersion5, socksMethodNoAuth})
			return err
		}
	}

	_, _ = conn.Write([]byte{socksVersion5, socksMethodNoAcceptable})
	return errors.New("socks5 client did not offer no-auth method")
}

func readSOCKS5ConnectRequest(conn net.Conn) (string, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", fmt.Errorf("read socks5 request header: %w", err)
	}
	if header[0] != socksVersion5 {
		return "", fmt.Errorf("unsupported socks request version %d", header[0])
	}
	if header[1] != socksCmdConnect {
		return "", &socksRequestError{
			reply: socksReplyCommandUnsupported,
			err:   fmt.Errorf("unsupported socks command %d", header[1]),
		}
	}

	host, err := readSOCKS5Address(conn, header[3])
	if err != nil {
		return "", &socksRequestError{reply: socksReplyAddrUnsupported, err: err}
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return "", fmt.Errorf("read socks5 target port: %w", err)
	}
	port := int(binary.BigEndian.Uint16(portBytes))

	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

type socksRequestError struct {
	reply byte
	err   error
}

func (e *socksRequestError) Error() string {
	return e.err.Error()
}

func (e *socksRequestError) Unwrap() error {
	return e.err
}

func readSOCKS5Address(conn net.Conn, atyp byte) (string, error) {
	switch atyp {
	case socksAddrIPv4:
		buf := make([]byte, net.IPv4len)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", fmt.Errorf("read socks5 ipv4 address: %w", err)
		}
		return net.IP(buf).String(), nil
	case socksAddrIPv6:
		buf := make([]byte, net.IPv6len)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", fmt.Errorf("read socks5 ipv6 address: %w", err)
		}
		return net.IP(buf).String(), nil
	case socksAddrDomain:
		length := []byte{0}
		if _, err := io.ReadFull(conn, length); err != nil {
			return "", fmt.Errorf("read socks5 domain length: %w", err)
		}
		if length[0] == 0 {
			return "", errors.New("socks5 domain is empty")
		}
		buf := make([]byte, int(length[0]))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", fmt.Errorf("read socks5 domain: %w", err)
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported socks5 address type %d", atyp)
	}
}

func writeSOCKS5Reply(conn net.Conn, reply byte, bindAddr net.Addr) error {
	response := []byte{socksVersion5, reply, 0x00, socksAddrIPv4, 0, 0, 0, 0, 0, 0}

	if tcpAddr, ok := bindAddr.(*net.TCPAddr); ok && tcpAddr != nil {
		if ipv4 := tcpAddr.IP.To4(); ipv4 != nil {
			copy(response[4:8], ipv4)
		} else if ipv6 := tcpAddr.IP.To16(); ipv6 != nil {
			response = make([]byte, 4+net.IPv6len+2)
			response[0] = socksVersion5
			response[1] = reply
			response[2] = 0x00
			response[3] = socksAddrIPv6
			copy(response[4:20], ipv6)
			binary.BigEndian.PutUint16(response[20:22], uint16(tcpAddr.Port))
			_, err := conn.Write(response)
			return err
		}
		binary.BigEndian.PutUint16(response[8:10], uint16(tcpAddr.Port))
	}

	_, err := conn.Write(response)
	return err
}
