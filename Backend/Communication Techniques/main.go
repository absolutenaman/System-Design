package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	server := gin.Default()
	server.Handle("GET", "/getData", HandleShortPolling)
	server.Handle("GET", "/UpdateData", UpdatePollingData)
	err := server.Run("localhost:8000")
	if err != nil {
		panic(err)
	}
}

var data string

func HandleShortPolling(ctx *gin.Context) {
	data = "initial data"
	lastName := ctx.Query("lastName")
	fmt.Println(lastName)
	if lastName != "" {
	} else {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.JSON(http.StatusOK, gin.H{"data": data})
	}
}
func UpdatePollingData(ctx *gin.Context) {
	data = "Updated Data"
	ctx.Header("Access-Control-Allow-Origin", "*")

	ctx.JSON(http.StatusOK, gin.H{"data": data})
}
