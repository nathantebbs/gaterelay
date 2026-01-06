package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Relay manages the TCP relay service
type Relay struct {
	cfg      *Config
	logger   *slog.Logger
	listener net.Listener

	activeConns atomic.Int32
	totalConns  atomic.Uint64
	bytesRx     atomic.Uint64
	bytesTx     atomic.Uint64

	wg       sync.WaitGroup
	shutdown chan struct{}
}

// NewRelay creates a new relay service
func NewRelay(cfg *Config, logger *slog.Logger) *Relay {
	return &Relay{
		cfg:      cfg,
		logger:   logger,
		shutdown: make(chan struct{}),
	}
}

// Start begins listening and accepting connections
func (r *Relay) Start(ctx context.Context) error {
	listenAddr := fmt.Sprintf("%s:%d", r.cfg.ListenAddr, r.cfg.ListenPort)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}
	r.listener = listener

	r.logger.Info("relay started",
		"listen_addr", listenAddr,
		"target_addr", fmt.Sprintf("%s:%d", r.cfg.TargetAddr, r.cfg.TargetPort),
		"max_conns", r.cfg.MaxConns)

	go r.acceptLoop(ctx)

	return nil
}

// acceptLoop continuously accepts new connections
func (r *Relay) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.shutdown:
			return
		default:
		}

		conn, err := r.listener.Accept()
		if err != nil {
			select {
			case <-r.shutdown:
				return
			default:
				r.logger.Error("accept failed", "error", err)
				continue
			}
		}

		// Check connection limit
		active := r.activeConns.Load()
		if active >= int32(r.cfg.MaxConns) {
			r.logger.Warn("connection limit reached, rejecting",
				"active_conns", active,
				"max_conns", r.cfg.MaxConns,
				"remote_addr", conn.RemoteAddr().String())
			conn.Close()
			continue
		}

		r.wg.Add(1)
		go r.handleConnection(conn)
	}
}

// handleConnection manages a single client connection
func (r *Relay) handleConnection(clientConn net.Conn) {
	defer r.wg.Done()

	r.activeConns.Add(1)
	defer r.activeConns.Add(-1)

	connID := r.totalConns.Add(1)
	clientAddr := clientConn.RemoteAddr().String()

	logger := r.logger.With(
		"conn_id", connID,
		"client_addr", clientAddr,
	)

	logger.Info("connection accepted")
	defer clientConn.Close()

	// Set idle timeout if configured
	if r.cfg.IdleTimeoutSecs > 0 {
		timeout := time.Duration(r.cfg.IdleTimeoutSecs) * time.Second
		clientConn.SetDeadline(time.Now().Add(timeout))
	}

	// Connect to target
	targetAddr := fmt.Sprintf("%s:%d", r.cfg.TargetAddr, r.cfg.TargetPort)

	dialer := net.Dialer{}
	if r.cfg.ConnectTimeoutSecs > 0 {
		dialer.Timeout = time.Duration(r.cfg.ConnectTimeoutSecs) * time.Second
	}

	targetConn, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		logger.Error("failed to connect to target",
			"target_addr", targetAddr,
			"error", err)
		return
	}
	defer targetConn.Close()

	logger.Info("connected to target", "target_addr", targetAddr)

	// Set idle timeout on target connection
	if r.cfg.IdleTimeoutSecs > 0 {
		timeout := time.Duration(r.cfg.IdleTimeoutSecs) * time.Second
		targetConn.SetDeadline(time.Now().Add(timeout))
	}

	// Bidirectional relay
	var bytesRx, bytesTx int64
	done := make(chan error, 2)

	// Client -> Target
	go func() {
		n, err := io.Copy(targetConn, clientConn)
		bytesRx = n
		done <- err
	}()

	// Target -> Client
	go func() {
		n, err := io.Copy(clientConn, targetConn)
		bytesTx = n
		done <- err
	}()

	// Wait for one direction to complete
	err = <-done

	// Close connections to terminate the other direction
	clientConn.Close()
	targetConn.Close()

	// Wait for second direction
	<-done

	// Update global counters
	r.bytesRx.Add(uint64(bytesRx))
	r.bytesTx.Add(uint64(bytesTx))

	logger.Info("connection closed",
		"bytes_rx", bytesRx,
		"bytes_tx", bytesTx,
		"error", err)
}

// Shutdown gracefully stops the relay service
func (r *Relay) Shutdown(timeout time.Duration) error {
	r.logger.Info("shutdown initiated", "timeout_secs", timeout.Seconds())

	close(r.shutdown)

	if r.listener != nil {
		r.listener.Close()
	}

	// Wait for active connections with timeout
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Info("all connections closed gracefully")
	case <-time.After(timeout):
		r.logger.Warn("shutdown timeout reached, forcing close",
			"active_conns", r.activeConns.Load())
	}

	r.logger.Info("shutdown complete",
		"total_conns", r.totalConns.Load(),
		"bytes_rx", r.bytesRx.Load(),
		"bytes_tx", r.bytesTx.Load())

	return nil
}

// Stats returns current relay statistics
func (r *Relay) Stats() map[string]interface{} {
	return map[string]interface{}{
		"active_conns": r.activeConns.Load(),
		"total_conns":  r.totalConns.Load(),
		"bytes_rx":     r.bytesRx.Load(),
		"bytes_tx":     r.bytesTx.Load(),
	}
}
