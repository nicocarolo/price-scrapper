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

func Add(c *gin.Context) {
	var merchant models.Merchant
	err := c.BindJSON(&merchant)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot resolve request: " + err.Error(),
		})
		return
	}

	session, err := db.GetMongoSession()
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Can't connect to database",
		})
		return
	}
	defer db.CloseMongoSession(session)

	collection := session.DB("heroku_rjnls62m").C("merchants")

	var results []models.Merchant
	err = collection.Find(bson.M{
		"name":                    merchant.Name,
		"products_url":            merchant.Products_url,
		"subsidiarys_url":         merchant.Subsidiarys_url,
		"products_id_selector":    merchant.Products_id_selector,
		"products_name_selector":  merchant.Products_name_selector,
		"products_price_selector": merchant.Products_price_selector,
	}).All(&results)

	if len(results) > 0 {
		log.Println(fmt.Sprintf("Merchant %s exists", merchant.Name))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Merchant already exists",
		})
		return
	}

	merchant.Id = bson.NewObjectId()
	err = collection.Insert(&merchant)
	if err != nil {
		log.Fatal(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot create merchant",
		})
		return
	}

	productsQuantity := collectProducts(merchant.Id, session)
	if productsQuantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot get products",
		})
		collection.Remove(bson.M{"_id": merchant.Id})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Added merchant",
		"products": productsQuantity,
	})
}
