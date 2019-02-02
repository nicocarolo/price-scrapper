package models

type Price struct {
	Merchant string
	Value    float64
}

type Product struct {
	Sku    string
	Name   string
	Prices map[string]Price
}
