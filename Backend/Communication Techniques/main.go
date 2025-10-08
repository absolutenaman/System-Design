package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"sync"
)

var (
	currentMessage string = "Initial message"
	messageID      int    = 0
	mu             sync.Mutex
	cond           *sync.Cond
)

func init() {
	cond = sync.NewCond(&mu)
}

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true

	r.Use(cors.New(config))
	r.POST("/message", postMessageHandler)
	r.GET("/poll", pollHandler)
	log.Println("Server running on http://localhost:8080")
	r.Run("localhost:8080")
}

func postMessageHandler(c *gin.Context) {

	var json struct {
		Message string `json:"message" binding:"required"`
	}

	if c.BindJSON(&json) == nil {
		mu.Lock()
		currentMessage = json.Message
		messageID++
		mu.Unlock()
		cond.Broadcast()
		c.JSON(http.StatusOK, gin.H{"status": "updated"})
	}
}

func pollHandler(c *gin.Context) {

	clientVersion := 0
	messageID = 0
	mu.Lock()
	if messageID > clientVersion {
		response := gin.H{"message": currentMessage, "version": messageID}
		mu.Unlock()
		c.JSON(http.StatusOK, response)
		return
	}

	done := make(chan struct{})

	go func() {
		cond.Wait()
		close(done)
	}()

	select {
	case <-done:
		response := gin.H{"message": currentMessage, "version": messageID}
		mu.Unlock()

		c.JSON(http.StatusOK, response)

	case <-c.Request.Context().Done():
		mu.Unlock()

		c.Status(http.StatusNoContent)
	}
}
