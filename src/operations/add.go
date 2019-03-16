package operations

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gocolly/colly"
	"github.com/pretty"
	"github.com/price-scrapper/src/config"
	"github.com/price-scrapper/src/db"
	"github.com/price-scrapper/src/models"
	"googlemaps.github.io/maps"
	"log"
	"math/rand"
	"net/http"
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

	collection := session.DB(config.PriceDB).C(config.MerchantsCollection)

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
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot create merchant",
		})
		return
	}

	productsQuantity, collectError := collectProducts(merchant.Id, session)
	//productsQuantity := 14
	if collectError != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"products_error": collectError.Error(),
		})
	} else {
		if productsQuantity > 0 {
			subsidiarysQuantity, subsError := collectSubsidiarys(merchant, session)
			if subsError != nil {
				c.JSON(http.StatusOK, gin.H{
					"message":           "Added merchant",
					"products":          productsQuantity,
					"subsidiarys":       subsidiarysQuantity,
					"subsidiarys_error": subsError.Error(),
				})
			} else {
				if subsidiarysQuantity > 0 {
					c.JSON(http.StatusOK, gin.H{
						"message":     "Added merchant",
						"products":    productsQuantity,
						"subsidiarys": subsidiarysQuantity,
					})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{
						"subsidiarys_error": "Cannot found subsidiarys",
					})
					collection.Remove(bson.M{"_id": merchant.Id})
				}
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"products_error": "Cannot found products",
			})
			collection.Remove(bson.M{"_id": merchant.Id})
		}
	}
}

func collectSubsidiarys(merchant models.Merchant, session *mgo.Session) (int, error) {
	log.Println("Starting to decide subsidiarys method")
	var subsidiarysQuantity int
	var err error
	if merchant.Subsidiarys_url != "" {
		subsidiarysQuantity, err = SubsidiarysByScrapping(session, merchant)
	} else {
		subsidiarysQuantity, err = SubsidiarysByGoogle(session, merchant)
	}

	log.Println("Finish to collect subsidiarys method")
	return subsidiarysQuantity, err
}

func SubsidiarysByGoogle(session *mgo.Session, merchant models.Merchant) (int, error) {
	log.Println("Starting to collect subsidiarys with Google Places")
	collection := session.DB(config.PriceDB).C(config.MerchantsCollection)
	subsidiarysQuantity := 0
	client, err := maps.NewClient(maps.WithAPIKey(config.GoogleApiKey))
	if err != nil {
		log.Println(fmt.Errorf("Error while post to google api: %s", err.Error()))
		return subsidiarysQuantity, fmt.Errorf("Cannot resolve request: %s", err.Error())
	}

	r := &maps.NearbySearchRequest{
		Radius:   50000,
		Location: &maps.LatLng{Lat: -34.546573, Lng: -58.493247},
		Keyword:  merchant.Name,
	}

	resp, err := client.NearbySearch(context.Background(), r)
	var subs []models.Location

	for len(resp.Results) > 0 && err == nil {
		for _, result := range resp.Results {
			if Contains(result.Types, "supermarket") || Contains(result.Types, "grocery_or_supermarket") {
				subs = append(subs, models.Location{Lat: fmt.Sprintf("%f", result.Geometry.Location.Lat), Lng: fmt.Sprintf("%f", result.Geometry.Location.Lng)})
				subsidiarysQuantity++
			}
		}
		if resp.NextPageToken != "" {
			r = &maps.NearbySearchRequest{
				Radius:    50000,
				Location:  &maps.LatLng{Lat: -34.546573, Lng: -58.493247},
				Keyword:   merchant.Name,
				PageToken: resp.NextPageToken,
			}
			resp, err = client.NearbySearch(context.Background(), r)
		} else {
			resp.Results = resp.Results[:0]
		}
	}
	if subsidiarysQuantity > 0 {
		collection.Update(bson.M{"_id": merchant.Id}, bson.M{"$set": bson.M{"subsidiarys": subs}})
	} else {
		if err != nil {
			err = fmt.Errorf("Cannot find subsidiarys on Places: %s", err.Error())
		} else {
			err = fmt.Errorf("Cannot find subsidiarys on Places")
		}
		return subsidiarysQuantity, err
	}
	log.Println("Finished collect subsidiarys with Google Places")
	return subsidiarysQuantity, nil
}

func SubsidiarysByScrapping(session *mgo.Session, merchant models.Merchant) (int, error) {
	log.Println("Finished collect subsidiarys with Scrapping")
	collection := session.DB(config.PriceDB).C(config.MerchantsCollection)
	subsidiarysQuantity := 0
	client, err := maps.NewClient(maps.WithAPIKey(config.GoogleApiKey))
	if err != nil {
		log.Println(fmt.Errorf("Error while post to google api: %s", err.Error()))
		return subsidiarysQuantity, err
	}

	var url string
	// begin := time.Now()

	collector := colly.NewCollector()

	// collector.OnHTML(merchant.Subsidiarys_container_selector, func(e *colly.HTMLElement) {
	collector.OnHTML(merchant.Subsidiarys_selector, func(e *colly.HTMLElement) {
		address := e.Text
		log.Println("Finding subsidiary address: %s", address)

		r := &maps.GeocodingRequest{
			Address: address,
		}

		resp, err := client.Geocode(context.Background(), r)

		if err == nil && len(resp) > 0 {
			result := resp[0]
			pretty.Println(resp)
			var m models.Merchant
			err := collection.Find(bson.M{"_id": merchant.Id, "subsidiarys.lat": result.Geometry.Location.Lat, "subsidiarys.lng": result.Geometry.Location.Lng}).One(&m)

			if err != nil {
				collection.Update(bson.M{"_id": merchant.Id}, bson.M{"$push": bson.M{"subsidiarys": models.Location{Address: result.FormattedAddress, Lat: fmt.Sprintf("%f", result.Geometry.Location.Lat), Lng: fmt.Sprintf("%f", result.Geometry.Location.Lng)}}})
				subsidiarysQuantity++
			}
		} else {
			if err != nil {
				log.Println(fmt.Errorf("Error on Google Geocode for address: %s, result: %s", address, err.Error()))
			} else {
				log.Println("There is no result from Google geocode for address: %s", address)
			}

		}
	})

	// Before making a request put the URL with
	// the key of "url" into the context of the request
	collector.OnRequest(func(r *colly.Request) {
		randInt := rand.Intn(len(UserAgents) - 1)
		r.Headers.Set("User-Agent", UserAgents[randInt])

		log.Println("Visitando: %s", r.URL.String())
		r.Ctx.Put("url", r.URL.String())
	})

	// After making a request get "url" from
	// the context of the request
	collector.OnResponse(func(r *colly.Response) {
		url = r.Ctx.Get("url")
		log.Println(r.StatusCode)
	})

	// Start scraping
	err = collector.Visit(merchant.Subsidiarys_url)
	if err != nil {
		log.Println(err.Error())
	}

	collector.Wait()
	return subsidiarysQuantity, nil
}

func Contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}
