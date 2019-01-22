package operations

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/gocolly/colly"
	"github.com/price-scrapper/src/db"
	"github.com/price-scrapper/src/models"
)

func Process(c *gin.Context) {
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

	collector := colly.NewCollector(
		colly.URLFilters(
			regexp.MustCompile("https://www.jumbo.com.ar/*"),
		),
		// MaxDepth is 2, so only the links on the scraped page
		// and links on those pages are visited
		colly.MaxDepth(4),
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

		result := models.Product{}
		result.Prices = append(result.Prices, models.Price{
			Merchant: "prueba1",
			Value:    price,
		})

		err = collection.Find(bson.M{"sku": sku}).One(&result)

		if err != nil {
			log.Fatal(err)
			err = collection.Insert(&models.Product{Sku: sku, Name: name, Prices: result.Prices})
		}

		err = collection.Update(bson.M{"sku": result.Sku}, result)
		if err != nil {
			log.Fatal(err)

		}

	})

	// Before making a request print "Visiting ..."
	collector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping on https://hackerspaces.org
	collector.Visit("https://www.jumbo.com.ar/almacen")

	collector.Wait()

	c.JSON(http.StatusOK, gin.H{
		"message": "process",
	})
}
