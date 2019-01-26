package models

import "github.com/globalsign/mgo/bson"

type Merchant struct {
	Id                      bson.ObjectId `json:"id"        bson:"_id,omitempty"`
	Name                    string        `json:"merchant" binding:"required"`
	Subsidiarys_url         string        `json:"subsidiarys_url" binding:"required"`
	Subsidiarys_selector    string        `json:"subsidiarys_selector" binding:"required"`
	Products_url            string        `json:"products_url" binding:"required"`
	Products_id_selector    string        `json:"products_id_selector" binding:"required"`
	Products_name_selector  string        `json:"products_name_selector" binding:"required"`
	Products_price_selector string        `json:"products_price_selector" binding:"required"`
	Price_offset            int32         `json:"price_offset" binding:"required"`
	Price_decimal_separator string        `json:"price_decimal_separator" binding:"required"`
}
