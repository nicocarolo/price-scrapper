package models

type Price struct {
	Merchant string
	Value    string
}

type Product struct {
	Sku    string
	Name   string
	Prices map[string]Price
}
