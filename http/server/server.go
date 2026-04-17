// Package server re-exports the top-level server package at the http/server import path.
// Services import this package as: srv "github.com/rdevitto86/komodo-forge-sdk-go/http/server"
package server

import (
	"net/http"
	"time"

	komodoserver "github.com/rdevitto86/komodo-forge-sdk-go/server"
)

// Run starts the HTTP server on port and blocks until the process receives a shutdown signal,
// then performs a graceful drain up to gracefulTimeout.
func Run(srv *http.Server, port string, gracefulTimeout time.Duration) {
	komodoserver.Run(srv, port, gracefulTimeout)
}

// InitAndServe is an alias for Run retained for backward compatibility.
func InitAndServe(srv *http.Server, p string, t time.Duration) {
	komodoserver.InitAndServe(srv, p, t)
}
