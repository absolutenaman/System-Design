package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func main() {
	server := gin.Default()
	server.Handle("GET", "/getData", SendStreamOfEvents)
	err := server.Run("localhost:8000")
	if err != nil {
		panic(err)
	}
}

func SendStreamOfEvents(ctx *gin.Context) {
	var number int
	number = 0
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")
	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		http.Error(ctx.Writer, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	fmt.Fprintf(ctx.Writer, "data: Welcome to server side event\n\n")
	flusher.Flush()
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		time.Sleep(100 * time.Second)
		done <- true
	}()
	for {
		select {
		case <-done:
			fmt.Fprintf(ctx.Writer, "event: done\ndata: Stream ended\n\n")
			flusher.Flush()
			return
		case _ = <-ticker.C:
			msg := fmt.Sprintf("Current time: %v", number)
			number += 1
			fmt.Fprintf(ctx.Writer, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
