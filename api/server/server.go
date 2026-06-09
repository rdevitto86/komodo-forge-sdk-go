package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
	"net/http"
)

// Default server timeouts applied by InitAndServe when the caller leaves a field zero.
// ReadHeaderTimeout in particular guards against Slowloris (slow-header) attacks.
var (
	DefaultReadHeaderTimeout = 10 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultWriteTimeout      = 30 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
)

// Applies the package default timeouts to any field the caller left at its zero value,
// hardening the server against slow-client (Slowloris) resource exhaustion.
func applyTimeoutDefaults(srv *http.Server) {
	if srv.ReadHeaderTimeout == 0 {
		srv.ReadHeaderTimeout = DefaultReadHeaderTimeout
	}
	if srv.ReadTimeout == 0 {
		srv.ReadTimeout = DefaultReadTimeout
	}
	if srv.WriteTimeout == 0 {
		srv.WriteTimeout = DefaultWriteTimeout
	}
	if srv.IdleTimeout == 0 {
		srv.IdleTimeout = DefaultIdleTimeout
	}
}

// Runs the server as a Lambda function when AWS_LAMBDA_FUNCTION_NAME is set, otherwise falls through to InitAndServe.
func Run(srv *http.Server, port string, gracefulTimeout time.Duration) {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(httpadapter.NewV2(srv.Handler).ProxyWithContext)
		return
	}
	InitAndServe(srv, port, gracefulTimeout)
}

// Starts the server on port p and blocks until a shutdown signal is received, then drains within timeout t.
func InitAndServe(srv *http.Server, p string, t time.Duration) {
	// Create context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv.Addr = p              // Set server address
	applyTimeoutDefaults(srv) // Harden against slow-client DoS when caller omits timeouts

	// Start server in goroutine
	go func() {
		logger.Info("server starting", logger.Attr("addr", p))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to start", err)
			stop() // Cancel the context to unblock main goroutine
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("server shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()

	// Gracefully shutdown server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", err)
		os.Exit(1)
	}

	logger.Info("server shutdown complete")
}
