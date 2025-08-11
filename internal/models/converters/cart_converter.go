package converters

import (
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/google/uuid"
)

func MapToCartResponse(userID uuid.UUID, cart *models.CartItem) *models.CartItemResponse {
	return &models.CartItemResponse{
		ID:          cart.ID,
		ProductID:   cart.ProductID,
		ProductName: cart.ProductName,
		Price:       cart.ProductPrice,
		UserID:      userID,
		Quantity:    cart.Quantity,
		Description: cart.CartDescription,
		CreatedAt:   cart.CreatedAt,
		UpdatedAt:   cart.UpdatedAt,
	}
}
