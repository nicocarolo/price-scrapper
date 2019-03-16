package config

import "os"

const PriceDB = "heroku_rjnls62m"
const MerchantsCollection = "merchants"
const ProductsCollection = "products"

const GoogleApiKey = "AIzaSyBoI5o-cwADaZTvIqalg08KkNH90OSjAPw"

const MaxTimeToWaitOnProducts = 1
const MaxTimeToScrap = 1

func GetURLDB() string {
	env := os.Getenv("ENVIRONMENT")
	var url string

	if env == "PRODUCTION" {
		url = "mongodb://api:dM6CYayNQu8qr9b@ds149984.mlab.com:49984/heroku_rjnls62m"
	} else {
		url = "localhost"
	}
	return url
}

func GetWebDiffURL() string {
	env := os.Getenv("ENVIRONMENT")
	var url string

	if env == "PRODUCTION" {
		url = "https://go-webdiff-job.herokuapp.com/%s"
	} else {
		url = "http://localhost:4000/%s"
	}
	return url
}
