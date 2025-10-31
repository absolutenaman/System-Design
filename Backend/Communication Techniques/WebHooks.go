package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// WebhookPayload is a generic struct to unmarshal the incoming JSON payload.
// In a real scenario, you'd define this based on the specific webhook provider (e.g., GitHub, Stripe).
type WebhookPayload struct {
	EventType string                 `json:"event_type"` // Example field; adjust as needed
	Data      map[string]interface{} `json:"data"`       // Generic data field
}

var (
	clients = make(map[chan string]bool) // Track SSE clients
	mutex   sync.Mutex                   // For thread-safe broadcasting
)

// Broadcast function: Send event to all connected clients
func broadcast(eventType, message string) {
	mutex.Lock()
	defer mutex.Unlock()
	payload := gin.H{"event": eventType, "data": message}
	jsonData, _ := json.Marshal(payload)
	msg := jsonData

	for client := range clients {
		select {
		case client <- string(msg):
		default:
			delete(clients, client)
		}
	}
}

func webhookHandler(c *gin.Context) {
	// Ensure it's a POST request (Gin handles routing, but we can still check)
	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		return
	}

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}
	c.Request.Body.Close() // Explicitly close, though Gin handles it

	// Log the raw payload for debugging (in production, log securely)
	fmt.Printf("Received webhook payload: %s\n", string(body))

	// Unmarshal into a struct (adjust based on expected format)
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Bind JSON directly (alternative to manual unmarshal; use if payload matches exactly)
	// if err := c.ShouldBindJSON(&payload); err != nil { ... }

	// Process the payload (example: print event details)
	fmt.Printf("Event Type: %s\n", payload.EventType)
	for key, value := range payload.Data {
		fmt.Printf("Data[%s]: %v\n", key, value)
	}

	// Process: Simulate saving to DB and get a message
	message := fmt.Sprintf("New event: %s for user %v", payload.EventType, payload.Data["user_id"])

	// Broadcast to frontend
	broadcast("notification", message)

	// In a real app, you'd validate signatures, update databases, send notifications, etc.
	// For now, just acknowledge success

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func sseHandler(c *gin.Context) {
	// SSE endpoint for frontend to connect
	client := make(chan string)
	mutex.Lock()
	clients[client] = true
	mutex.Unlock()
	defer func() {
		mutex.Lock()
		delete(clients, client)
		mutex.Unlock()
		close(client)
	}()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	for {
		select {
		case msg := <-client:
			c.SSEvent("", msg)
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

//func main() {
//	// Set up Gin router
//	r := gin.Default()
//
//	// Webhook endpoint (receives from upstream like Stripe)
//	r.POST("/webhook", webhookHandler)
//
//	// SSE endpoint (downstream API that "tells" frontend)
//	r.GET("/events", sseHandler)
//
//	port := ":8080"
//	fmt.Printf("Server on http://localhost%s (webhook: /webhook, events: /events)\n", port)
//	r.Run(port)
//}
