package websocket

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

const (
	// maxMessageBytes caps a single inbound frame so a peer cannot exhaust server memory.
	maxMessageBytes = 1 << 20 // 1 MiB
	// readTimeout bounds how long a read may block before the peer must send data or a pong.
	readTimeout = 60 * time.Second
	// writeTimeout bounds a single write so a slow reader cannot stall the writer.
	writeTimeout = 10 * time.Second
	// handshakeTimeout bounds the upgrade handshake itself.
	handshakeTimeout = 10 * time.Second
)

var (
	allowedOriginsMu sync.RWMutex
	allowedOrigins   = map[string]struct{}{}
)

// Replaces the WebSocket upgrade origin allowlist; an empty list rejects every cross-origin upgrade.
func SetAllowedOrigins(origins []string) {
	m := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		m[strings.ToLower(o)] = struct{}{}
	}
	allowedOriginsMu.Lock()
	allowedOrigins = m
	allowedOriginsMu.Unlock()
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	allowedOriginsMu.RLock()
	defer allowedOriginsMu.RUnlock()
	_, ok := allowedOrigins[strings.ToLower(origin)]
	return ok
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: handshakeTimeout,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin:      checkOrigin,
}

type Client struct {
	conn *websocket.Conn
}

// Upgrades the HTTP request to a WebSocket connection with hardened defaults (origin
// allowlist, bounded message size, read/write deadlines) and returns a Client the caller
// drives. The caller owns the connection lifecycle and must call Close.
func Upgrade(wtr http.ResponseWriter, req *http.Request) (*Client, error) {
	conn, err := upgrader.Upgrade(wtr, req, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", err)
		return nil, err
	}
	conn.SetReadLimit(maxMessageBytes)
	_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(readTimeout))
	})
	return &Client{conn: conn}, nil
}

// Wraps an existing WebSocket connection in a Client; prefer Upgrade, which also applies
// the hardened read limit and deadlines.
func NewClient(conn *websocket.Conn) *Client {
	return &Client{conn: conn}
}

// Sends a message, bounding the write with a deadline so a slow reader cannot stall the writer.
func (c *Client) Write(messageType int, data []byte) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return c.conn.WriteMessage(messageType, data)
}

// Reads a message and extends the read deadline on success.
func (c *Client) Read() (int, []byte, error) {
	mt, data, err := c.conn.ReadMessage()
	if err == nil {
		_ = c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	}
	return mt, data, err
}

// Closes the WebSocket connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
