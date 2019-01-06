package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/price-scrapper/src/operations"
)

func main() {
	fmt.Println("Hello world")

	r := gin.Default()

	r.GET("/ping", operations.Ping)

	r.POST("/add", operations.Add)

	r.POST("/process", operations.Process)

	r.Run(":3000")
}
