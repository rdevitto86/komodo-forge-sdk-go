package websocket

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	allowedOriginsMu sync.RWMutex
	allowedOrigins   = map[string]struct{}{}
)

// SetAllowedOrigins replaces the WebSocket upgrade origin allowlist; an empty list rejects every cross-origin upgrade.
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
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

type Client struct {
	conn *websocket.Conn
}

// Creates a new WebSocket client from an existing connection.
func NewClient(conn *websocket.Conn) *Client {
	return &Client{conn: conn}
}

// Sends a message to the WebSocket connection.
func (c *Client) Write(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

// Reads a message from the WebSocket connection.
func (c *Client) Read() (int, []byte, error) {
	return c.conn.ReadMessage()
}

// Closes the WebSocket connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Upgrades the HTTP connection to WebSocket and echoes received messages back to the client.
func RouteHandler(wtr http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(wtr, req, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	client := NewClient(conn)

	for {
		messageType, message, err := client.Read()
		if err != nil {
			log.Printf("websocket read error: %v", err)
			break
		}

		log.Printf("received message: %s", message)

		if err := client.Write(messageType, message); err != nil {
			log.Printf("websocket write error: %v", err)
			break
		}
	}
}
