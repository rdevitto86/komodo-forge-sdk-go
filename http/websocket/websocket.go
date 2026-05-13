package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
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

// Route handler that upgrades HTTP connections to WebSocket and handles messages.
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
