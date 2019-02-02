package operations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gocolly/colly"
	"github.com/price-scrapper/src/config"
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

	productsQuantity, collectError := collectProducts(bson.NewObjectId(), session)
	if collectError != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": collectError.Error(),
		})
	} else {
		if productsQuantity > 0 {
			c.JSON(http.StatusOK, gin.H{
				"message":  "process",
				"products": productsQuantity,
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Cannot found products",
			})
		}
	}
}

func collectProducts(id bson.ObjectId, session *mgo.Session) (int, error) {
	collection := session.DB(config.PriceDB).C(config.MerchantsCollection)
	var merchant models.Merchant

	err := collection.Find(bson.M{"_id": id}).One(&merchant)

	if err != nil {
		return 0, fmt.Errorf("Cannot found merchant")
	}

	productsQuantity := 0
	collection = session.DB(config.PriceDB).C(config.ProductsCollection)

	var url string

	collector := colly.NewCollector(
		colly.URLFilters(
			regexp.MustCompile(strings.Split(merchant.Products_url, ".com")[0]+"/*"),
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

	collector.OnHTML(merchant.Products_container_selector, func(e *colly.HTMLElement) {
		if e.DOM.Find(merchant.Products_id_selector).Length() == 0 {
			return
		}

		name := e.ChildText(merchant.Products_name_selector)
		sku := e.ChildText(merchant.Products_id_selector)
		str := e.ChildText(merchant.Products_price_selector)
		price, err := normalizePrice(str, ".", ",")

		if err != nil {
			log.Println(fmt.Errorf("Cannot parse price: %s", err.Error()))
			return
		}

		var result models.Product
		result.Prices = make(map[string]models.Price)

		err = collection.Find(bson.M{"sku": sku}).One(&result)

		result.Prices[id.Hex()] = models.Price{
			Merchant: id.Hex(),
			Value:    price,
		}

		if err != nil {
			log.Println("Insert product: %s: %s (%s)", sku, name, price)
			err = collection.Insert(&models.Product{Sku: sku, Name: name, Prices: result.Prices})
		} else {
			log.Println("Update product: %s: %s (%s)", sku, name, price)
			err = collection.Update(bson.M{"sku": result.Sku}, result)
		}
		if err != nil {
			log.Println(fmt.Errorf("Cannot insert or update product: %s", err.Error()))
		} else {
			productsQuantity++
		}

		values := map[string]string{"id": id.Hex(), "url": url}
		jsonValue, _ := json.Marshal(values)
		jobResponse, postErr := http.Post(fmt.Sprintf(config.GetWebDiffURL(), "/add"),
			"application/json", bytes.NewBuffer(jsonValue))
		defer jobResponse.Body.Close()
		if postErr != nil {
			log.Println(fmt.Errorf("Cannot post to job: %s", postErr.Error()))
		} else {
			body, _ := ioutil.ReadAll(jobResponse.Body)
			if jobResponse.StatusCode != http.StatusOK {
				log.Println(fmt.Errorf("Error while post to job: %s", string(body)))
			} else {
				log.Println("Saving url to job: %s", string(body))
			}
		}
	})

	// Before making a request put the URL with
	// the key of "url" into the context of the request
	collector.OnRequest(func(r *colly.Request) {
		log.Println(r.URL.String())
		r.Ctx.Put("url", r.URL.String())
	})

	// After making a request get "url" from
	// the context of the request
	collector.OnResponse(func(r *colly.Response) {
		url = r.Ctx.Get("url")
	})

	// Start scraping
	collector.Visit(merchant.Products_url)

	collector.Wait()
	return productsQuantity, nil
}

func normalizePrice(price string, kDelimiter string, decimalDelimiter string) (float64, error) {
	price = strings.Replace(price, " ", "", -1)
	price = strings.Replace(price, "$", "", -1)
	price = strings.Replace(price, kDelimiter, "", -1)
	price = strings.Replace(price, decimalDelimiter, ".", -1)
	return strconv.ParseFloat(price, 64)
}
