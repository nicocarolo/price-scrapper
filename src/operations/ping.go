package operations

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
)

func Ping(c *gin.Context) {
	url := "mongodb://api:dM6CYayNQu8qr9b@ds149984.mlab.com:49984/heroku_rjnls62m"
	session, err := mgo.Dial(url)
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		c.JSON(599, gin.H{
			"message": "Can't connect to database",
		})
		return
	}
	defer session.Close()

	c.JSON(200, gin.H{
		"message": "pong",
	})
}
