package operations

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gocolly/colly"
	"github.com/price-scrapper/src/db"
	"github.com/price-scrapper/src/models"
)

type processRequest struct {
	Name string `json:"merchant" binding:"required"`
}

func Process(c *gin.Context) {
	var request processRequest
	err := c.BindJSON(&request)

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

	productsQuantity := collectProducts(bson.NewObjectId(), session)
	if productsQuantity > 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":  "process",
			"products": productsQuantity,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot get products",
		})
	}

}

func collectProducts(id bson.ObjectId, session *mgo.Session) int {
	fmt.Println("empieza collect")
	productsQuantity := 0
	collection := session.DB("heroku_rjnls62m").C("products")

	collector := colly.NewCollector(
		colly.URLFilters(
			regexp.MustCompile("https://www.jumbo.com.ar/*"),
		),
		// MaxDepth is 2, so only the links on the scraped page
		// and links on those pages are visited
		colly.MaxDepth(2),
		colly.Async(true),
	)

	// On every a element which has href attribute call callback
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Visit link found on page
		// Only those links are visited which are in AllowedDomains
		collector.Visit(e.Request.AbsoluteURL(link))
	})

	collector.OnHTML(".product-info", func(e *colly.HTMLElement) {
		if e.DOM.Find(".skuReference").Length() == 0 {
			return
		}

		name := e.ChildText("h1.name")
		sku := e.ChildText("div.skuReference")
		price := e.ChildText("strong.skuBestPrice")

		fmt.Printf("Saving product: %s -> %s %s\n", sku, name, price)
		var result models.Product
		result.Prices = make(map[string]models.Price)

		err := collection.Find(bson.M{"sku": sku}).One(&result)
		fmt.Printf("Saving product: %s", result)

		// result.Prices = append(result.Prices, models.Price{
		// 	Merchant: id.String(),
		// 	Value:    price,
		// })
		result.Prices[id.Hex()] = models.Price{
			Merchant: id.Hex(),
			Value:    "999999",
		}

		if err != nil {
			err = collection.Insert(&models.Product{Sku: sku, Name: name, Prices: result.Prices})
		} else {
			err = collection.Update(bson.M{"sku": result.Sku}, result)
		}
		if err != nil {
			log.Fatal(err)
			fmt.Println("Error de producto")
		} else {
			productsQuantity++
		}
	})

	// Before making a request print "Visiting ..."
	collector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping
	collector.Visit("https://www.jumbo.com.ar/almacen/aceites-y-vinagres")

	collector.Wait()
	fmt.Println("Fin collect")
	return productsQuantity
}
