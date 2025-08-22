package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	{
		events := router.Group("/events")

		events.GET("/all")
		events.POST("/create", func(ctx *gin.Context) {
		})
	}


	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			jsonData
		})
	})

	router.Run(":8080")
}
