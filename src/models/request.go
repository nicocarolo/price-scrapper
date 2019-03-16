package models

type ProcessRequest struct {
	Name string `json:"merchant" binding:"required"`
}

type ProductRequest struct {
	Sku      string   `json:"sku" binding:"required"`
	Origin   Location `json:"origin,omitempty"`
	MtsLimit int      `json:"mts_limit,omitempty"`
}
