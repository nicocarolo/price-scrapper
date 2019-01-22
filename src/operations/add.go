package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type request struct {
	Merchant                string `json:"merchant" binding:"required"`
	Subsidiarys_url         string `json:"subsidiarys_url" binding:"required"`
	Subsidiarys_selector    string `json:"subsidiarys_selector" binding:"required"`
	Products_url            string `json:"products_url" binding:"required"`
	Products_id_selector    string `json:"products_id_selector" binding:"required"`
	Products_name_selector  string `json:"products_name_selector" binding:"required"`
	Products_price_selector string `json:"products_price_selector" binding:"required"`
	Price_offset            int32  `json:"price_offset" binding:"required"`
	Price_decimal_separator string `json:"price_decimal_separator" binding:"required"`
}

func Add(c *gin.Context) {
	var json request
	err := c.BindJSON(json)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot resolve request: " + err.Error(),
		})
		return
	}
	err = isValidRequest(json)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid request: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "add",
	})
}

func isValidRequest(request request) error {
	return nil
}
