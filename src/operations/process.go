package operations

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/gocolly/colly"
	"github.com/price-scrapper/src/models"
)

func Process(c *gin.Context) {
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

	session.SetSafe(&mgo.Safe{})

	collection := session.DB("heroku_rjnls62m").C("products")

	collector := colly.NewCollector(
		colly.URLFilters(
			regexp.MustCompile("https://www.jumbo.com.ar/*"),
		),
		// MaxDepth is 2, so only the links on the scraped page
		// and links on those pages are visited
		colly.MaxDepth(3),
		colly.Async(true),
	)

	// On every a element which has href attribute call callback
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Print link
		fmt.Printf("Link found: %q -> %s\n", e.Text, link)
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
		// Print link
		fmt.Printf("Name found: %q \n", name)
		fmt.Printf("sku found: %q \n", sku)
		fmt.Printf("price found: %q \n", price)

		err = collection.Insert(&models.Product{Sku: sku, Name: name})
	})

	// Before making a request print "Visiting ..."
	collector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping on https://hackerspaces.org
	collector.Visit("https://www.jumbo.com.ar/almacen")

	collector.Wait()

	// s := make(map[interface{}]interface{})
	// sm.Range(func(k, v interface{}) bool {
	// 	s[k] = v
	// 	return true
	// })
	// fmt.Println(s)

	c.JSON(200, gin.H{
		"message": "process",
	})
}
