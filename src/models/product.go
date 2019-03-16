package models

type Price struct {
	Merchant string
	Value    float64
}

type Product struct {
	Sku    string           `json:"sku,omitempty" binding:"required"`
	Name   string           `json:"name,omitempty" binding:"required"`
	Prices map[string]Price `json:"prices,omitempty" binding:"required"`
}
