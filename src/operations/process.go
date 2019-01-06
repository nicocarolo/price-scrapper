package operations

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly"
)

func Process(c *gin.Context) {
	var sm sync.Map

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

		sm.Store(sku, name+": "+price)
		result, ok := sm.Load(sku)
		if ok {
			fmt.Println(result.(string))
		} else {
			fmt.Println("value not found for key: " + sku)
		}

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
