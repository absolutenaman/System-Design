package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// --- Configuration ---
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// --- Hub (broadcasting) ---
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
			h.notifyAll(fmt.Sprintf("%s joined", c.name))
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				h.notifyAll(fmt.Sprintf("%s left", c.name))
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Slow/unresponsive client — remove it
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) notifyAll(msg string) {
	h.broadcast <- []byte(fmt.Sprintf("[system] %s", msg))
}

// --- Client ---
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	name string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		// Log incoming messages
		log.Printf("📨 Received from %s: %s", c.name, string(msg))

		// Attach sender name before broadcasting
		out := fmt.Sprintf("%s: %s", c.name, string(msg))
		c.hub.broadcast <- []byte(out)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// send queued messages in same websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// --- WebSocket upgrader ---
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for demo; lock this down in production.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// --- Websocket handler ---
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// name from query param ?name=...
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "anon"
	}
	// Log new connections
	log.Printf("🔌 New connection from: %s", name)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), name: name}
	client.hub.register <- client

	// start pumps
	go client.writePump()
	go client.readPump()
}

func main() {
	hub := newHub()
	go hub.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	addr := ":8080"
	fmt.Printf("Starting server at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
