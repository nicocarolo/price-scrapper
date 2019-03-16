package operations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gocolly/colly"
	"github.com/price-scrapper/src/config"
	"github.com/price-scrapper/src/db"
	"github.com/price-scrapper/src/models"
)

var UserAgents = []string{
	"Mozilla/5.0 (X11; Linux i686; rv:64.0) Gecko/20100101 Firefox/64.0",
	"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:64.0) Gecko/20100101 Firefox/64.0",
	"Opera/9.80 (X11; Linux i686; Ubuntu/14.10) Presto/2.12.388 Version/12.16",
	"Opera/9.80 (Macintosh; Intel Mac OS X 10.14.1) Presto/2.12.388 Version/12.16",
	"Opera/12.0(Windows NT 5.2;U;en)Presto/22.9.168 Version/12.00",
	"Opera/12.0(Windows NT 5.1;U;en)Presto/22.9.168 Version/12.00",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_3) AppleWebKit/537.75.14 (KHTML, like Gecko) Version/7.0.3 Safari/7046A194A",
	"Mozilla/5.0 (iPad; CPU OS 6_0 like Mac OS X) AppleWebKit/536.26 (KHTML, like Gecko) Version/6.0 Mobile/10A5355d Safari/8536.25",
	"Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US) AppleWebKit/532.2 (KHTML, like Gecko) ChromePlus/4.0.222.3 Chrome/4.0.222.3 Safari/532.2",
	"Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US) AppleWebKit/525.28.3 (KHTML, like Gecko) Version/3.2.3 ChromePlus/4.0.222.3 Chrome/4.0.222.3 Safari/525.28.3",
}

func Process(c *gin.Context) {
	var request models.ProcessRequest
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

	productsQuantity, collectError := collectProducts(bson.ObjectIdHex(request.Name), session)
	if collectError != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": collectError.Error(),
		})

		return
	}
	if productsQuantity > 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":  "process",
			"products": productsQuantity,
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": "Cannot found products",
	})
}

func collectProducts(id bson.ObjectId, session *mgo.Session) (int, error) {
	collection := session.DB(config.PriceDB).C(config.MerchantsCollection)
	var merchant models.Merchant


	if err := collection.Find(bson.M{"_id": id}).One(&merchant); err != nil {
		return 0, fmt.Errorf("Cannot found merchant")
	}


	productsQuantity := 0
	collection = session.DB(config.PriceDB).C(config.ProductsCollection)

	var url string

	begin := time.Now()
	lastScrap := time.Now()

	collector := colly.NewCollector(
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
		lastScrap = time.Now()
		//if e.DOM.Find(merchant.Products_id_selector).Length() == 0 {
		//	return
		//}

		name := e.ChildText(merchant.Products_name_selector)
		var sku string
		var strPrice string
		if strings.Contains(merchant.Products_id_selector, "|") {
			splits := strings.Split(merchant.Products_id_selector, "| attr=")
			attr := splits[len(splits) - 1]
			sku = e.ChildAttr(splits[0], attr)
		} else {
			sku = e.ChildText(merchant.Products_id_selector)
		}
		if strings.Contains(merchant.Products_price_selector, "|") {
			splits := strings.Split(merchant.Products_price_selector, "| attr=")
			attr := splits[len(splits) - 1]
			strPrice = e.ChildAttr(splits[0], attr)
		} else {
			strPrice = e.ChildText(merchant.Products_price_selector)
		}
		price, err := normalizePrice(strPrice, ".", ",")

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
			if err != nil {
				log.Println(fmt.Errorf("Cannot insert product %s error: %s", sku, err.Error()))
			}
		} else {
			log.Println("Update product: %s: %s (%s)", sku, name, price)
			err = collection.Update(bson.M{"sku": result.Sku}, result)
			if err != nil {
				log.Println(fmt.Errorf("Cannot update product %s error: %s", sku, err.Error()))
			}
		}
		if err != nil {
			log.Println(fmt.Errorf("Cannot insert or update product: %s", err.Error()))
		} else {
			productsQuantity++
		}

		values := map[string]string{"id": id.Hex(), "url": url}
		jsonValue, _ := json.Marshal(values)
		jobResponse, postErr := http.Post(fmt.Sprintf(config.GetWebDiffURL(), "add"),
			"application/json", bytes.NewBuffer(jsonValue))
		if postErr != nil {
			log.Println(fmt.Errorf("Error while post to job: %s", postErr.Error()))
		} else {
			body, _ := ioutil.ReadAll(jobResponse.Body)
			if jobResponse.StatusCode != http.StatusOK {
				log.Println(fmt.Errorf("Not succesful post to job: %s", string(body)))
			} else {
				log.Println("Saving url to job: %s", string(body))
			}
			postErr = jobResponse.Body.Close()
			if postErr != nil {
				log.Println(fmt.Errorf("Cannot close body: %s", postErr.Error()))
			}
		}
	})

	// Before making a request put the URL with
	// the key of "url" into the context of the request
	collector.OnRequest(func(r *colly.Request) {
		if !strings.Contains(r.URL.Host, merchant.Name){
			r.Abort()
		}
		now := time.Now()
		scraped := now.Sub(lastScrap).Minutes()
		elapsed := now.Sub(begin).Minutes()
		if scraped > config.MaxTimeToWaitOnProducts || elapsed > config.MaxTimeToScrap {
			log.Println("Stop for excedeed time to scrap")
			collector.OnHTMLDetach("a[href]")
		}

		randInt := rand.Intn(len(UserAgents) - 1)
		r.Headers.Set("User-Agent", UserAgents[randInt])
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

func normalizePrice(price, kDelimiter, decimalDelimiter string) (float64, error) {
	reg, err := regexp.Compile("[^0-9]+")
	if err != nil {
		log.Println(fmt.Errorf("Error creating regex for price: %s", err.Error()))
	}
	price = reg.ReplaceAllString(price, "")
	price = strings.Replace(price, " ", "", -1)
	price = strings.Replace(price, "$", "", -1)
	price = strings.Replace(price, kDelimiter, "", -1)
	price = strings.Replace(price, decimalDelimiter, ".", -1)
	return strconv.ParseFloat(price, 64)
}
