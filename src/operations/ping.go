package operations

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/price-scrapper/src/db"
	"github.com/price-scrapper/src/models"
)

func Ping(c *gin.Context) {
	session, err := db.GetMongoSession()
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		c.JSON(599, gin.H{
			"message": "Can't connect to database",
		})
		return
	}
	defer db.CloseMongoSession(session)

	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func Prueba(c *gin.Context) {
	session, err := db.GetMongoSession()
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Can't connect to database",
		})
		return
	}
	defer db.CloseMongoSession(session)

	collection := session.DB("heroku_rjnls62m").C("products")

	var result models.Product

	err = collection.Find(bson.M{"sku": "12440701009"}).One(&result)
	if err != nil {
		println("no es nil")
		log.Println(err)
	}
	result.Prices["id.Hex()"] = models.Price{
		Merchant: "id.Hex()",
		Value:    99.9,
	}
	println(result.Name)
	println(result.Sku)
	println(result.Prices)
	// result.Name = "Cerveza Corona Coronita 210 Ml actualizadoooo"
	err = collection.Update(bson.M{"sku": result.Sku}, result)
	if err != nil {
		log.Println(err)

	}
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
