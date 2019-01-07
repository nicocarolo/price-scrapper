package operations

import (
	"fmt"

	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
)

func Ping(c *gin.Context) {
	url := "mongodb://api:@dM6CYayNQu8qr9b.mlab.com:49984/heroku_rjnls62m"
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
