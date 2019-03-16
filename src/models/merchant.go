package models

import "github.com/globalsign/mgo/bson"

type Location struct {
	Address string `json:"address,omitempty" bson:"address,omitempty"`
	Lat     string `json:"lat,omitempty" bson:"lat,omitempty"`
	Lng     string `json:"lng,omitempty" bson:"lng,omitempty"`
}

type Merchant struct {
	Id                             bson.ObjectId `json:"id"        bson:"_id,omitempty"`
	Name                           string        `json:"merchant" binding:"required"`
	Subsidiarys_container_selector string        `json:"subsidiarys_container_selector,omitempty"`
	Subsidiarys_url                string        `json:"subsidiarys_url,omitempty"`
	Subsidiarys_selector           string        `json:"subsidiarys_selector,omitempty"`
	Products_url                   string        `json:"products_url,omitempty" binding:"required"`
	Products_container_selector    string        `json:"products_container_selector,omitempty" binding:"required"`
	Products_id_selector           string        `json:"products_id_selector,omitempty" binding:"required"`
	Products_name_selector         string        `json:"products_name_selector,omitempty" binding:"required"`
	Products_price_selector        string        `json:"products_price_selector,omitempty" binding:"required"`
	Price_offset                   int32         `json:"price_offset,omitempty" binding:"required"`
	Price_decimal_separator        string        `json:"price_decimal_separator,omitempty" binding:"required"`
	Subsidiarys                    []Location    `json:"subsidiarys,omitempty"`
}
