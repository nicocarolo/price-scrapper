package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/price-scrapper/src/operations"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	router := gin.New()
	router.Use(gin.Logger())

	router.GET("/ping", operations.Ping)

	router.POST("/add", operations.Add)

	router.POST("/process", operations.Process)

	router.Run(":" + port)
}
