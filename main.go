package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultConfigPath = "/etc/gaterelay/config.toml"
	version           = "1.0.0"
)

func main() {
	var (
		configPath  = flag.String("config", defaultConfigPath, "path to configuration file")
		showVersion = flag.Bool("version", false, "show version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("GateRelay v%s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(cfg.LogLevel)
	logger.Info("starting gaterelay",
		"version", version,
		"config", *configPath)

	// Create relay service
	relay := NewRelay(cfg, logger)

	// Start relay
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := relay.Start(ctx); err != nil {
		logger.Error("failed to start relay", "error", err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig.String())

	// Cancel context
	cancel()

	// Graceful shutdown
	shutdownTimeout := time.Duration(cfg.ShutdownGraceSecs) * time.Second
	if err := relay.Shutdown(shutdownTimeout); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("gaterelay stopped")
}

// setupLogger creates a structured logger with the specified level
func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}
