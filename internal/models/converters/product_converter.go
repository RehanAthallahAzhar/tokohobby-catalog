package converters

import (
	"time"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
)

func MapToProductResponse(product *models.ProductWithSeller) *models.ProductResponse {
	return &models.ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Price:       product.Price,
		Stock:       product.Stock,
		Discount:    product.Discount,
		Type:        product.Type,
		Description: product.Description,
		SellerID:    product.SellerID,
		SellerName:  product.SellerName,
		CreatedAt:   product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   product.UpdatedAt.Format(time.RFC3339),
	}
}
