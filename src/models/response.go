package models

import (
	"github.com/globalsign/mgo/bson"
)

type ProductResponse struct {
	Product     Product                      `json:"product,omitempty"`
	Origin      Location                     `json:"origin,omitempty"`
	Merchant    []ProductMerchantResponse    `json:"merchant,omitempty"`
	Destination []ProductDestinationResponse `json:"destination,omitempty"`
}

type ProductMerchantResponse struct {
	Id    bson.ObjectId `json:"id,omitempty"`
	Name  string        `json:"name,omitempty"`
	Price float64       `json:"price,omitempty"`
}

type ProductDestinationResponse struct {
	Location Location `json:"location,omitempty"`
	Distance int      `json:"distance,omitempty"`
}
