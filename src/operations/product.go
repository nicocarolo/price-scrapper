package operations

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pretty"
	"github.com/price-scrapper/src/config"
	"github.com/price-scrapper/src/db"
	"github.com/price-scrapper/src/models"
	"googlemaps.github.io/maps"
)

func Product(c *gin.Context) {
	var request models.ProductRequest
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
			"error": "Can't connect to database",
		})
		return
	}
	defer db.CloseMongoSession(session)

	products := session.DB(config.PriceDB).C(config.ProductsCollection)
	var product models.Product
	err = products.Find(bson.M{"sku": request.Sku}).One(&product)
	if err != nil {
		log.Println(fmt.Errorf("Cannot find requested product: %s", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot find requested product",
		})
		return
	}

	var response models.ProductResponse
	if request.Origin != (models.Location{}) {
		merchant, price, destinations, errBestPrices := getBestPriceByOrigin(session, product, request.Origin, request.MtsLimit)
		if errBestPrices != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": errBestPrices.Error(),
			})
			return
		}
		response.Merchant = append(response.Merchant, models.ProductMerchantResponse{Id: merchant.Id, Name: merchant.Name, Price: price})
		response.Product.Name = product.Name
		response.Product.Sku = product.Sku
		response.Origin = request.Origin
		response.Destination = destinations
	} else {
		merchants := session.DB(config.PriceDB).C(config.MerchantsCollection)
		for _, price := range product.Prices {
			var merchant models.Merchant
			err = merchants.Find(bson.M{"_id": bson.ObjectIdHex(price.Merchant)}).One(&merchant)
			response.Merchant = append(response.Merchant, models.ProductMerchantResponse{Id: merchant.Id, Name: merchant.Name, Price: price.Value})
		}
		response.Product.Name = product.Name
		response.Product.Sku = product.Sku
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func getDistances(client *maps.Client, origin models.Location, destination []models.Location) (*maps.DistanceMatrixResponse, error) {
	log.Println("Getting distances for: %s, %s, %s", origin.Address, origin.Lat, origin.Lng)
	var source []string
	var dest []string
	if origin.Address != "" {
		source = append(source, origin.Address)
	} else {
		source = append(source, fmt.Sprintf("%s,%s", origin.Lat, origin.Lng))
	}
	for _, location := range destination {
		if len(dest) <= 20 {
			dest = append(dest, fmt.Sprintf("%s,%s", location.Lat, location.Lng))
		}
	}

	r := &maps.DistanceMatrixRequest{
		Units:        maps.UnitsMetric,
		Origins:      source,
		Destinations: dest,
	}
	resp, err := client.DistanceMatrix(context.Background(), r)
	log.Println(fmt.Sprintf("Result from getting distances: %s", pretty.Sprint(resp)))
	return resp, err
}

func getBestPriceByOrigin(session *mgo.Session, product models.Product, origin models.Location, mtsLimit int) (models.Merchant, float64, []models.ProductDestinationResponse, error) {
	log.Println("Getting price by origin")
	merchants := session.DB(config.PriceDB).C(config.MerchantsCollection)

	var merchant models.Merchant
	var destinations []models.ProductDestinationResponse
	var minimum float64
	minimum = 9999999999999999

	client, err := maps.NewClient(maps.WithAPIKey(config.GoogleApiKey))
	if err != nil {
		log.Println(fmt.Errorf("Error while post to google api: %s", err.Error()))
		return merchant, minimum, destinations, fmt.Errorf("Cannot resolve request: %s", err.Error())
	}

	var bestResult models.Price

	for _, price := range product.Prices {
		if price.Value < minimum {
			bestResult.Merchant = price.Merchant
			bestResult.Value = price.Value
			minimum = price.Value
		}
	}

	err = merchants.Find(bson.M{"_id": bson.ObjectIdHex(bestResult.Merchant)}).One(&merchant)
	distanceResponse, distanceError := getDistances(client, origin, merchant.Subsidiarys)

	if distanceError != nil {
		log.Println(fmt.Errorf("Error on post to google api: %s", distanceError.Error()))
		return merchant, minimum, destinations, fmt.Errorf("Cannot get distances: %s", distanceError.Error())
	}
	elements := (*distanceResponse).Rows[0]
	distancesElements := elements.Elements

	for k, distance := range distancesElements {
		if distance.Status == "OK" && (mtsLimit >= distance.Distance.Meters || mtsLimit == 0) {
			destination := models.ProductDestinationResponse{Location: merchant.Subsidiarys[k], Distance: distance.Distance.Meters}
			destination.Location.Address = (*distanceResponse).DestinationAddresses[k]
			destinations = append(destinations, destination)
		}
	}
	return merchant, minimum, destinations, nil
}
