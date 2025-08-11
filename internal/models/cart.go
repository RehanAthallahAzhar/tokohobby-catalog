package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RedisCartItem struct {
	Quantity        int       `json:"quantity"`
	CartDescription string    `json:"cart_description "`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type CartItem struct {
	ID              uuid.UUID `json:"id"`
	SellerID        uuid.UUID `json:"seller_id"`
	SellerName      string    `json:"seller_name"`
	Quantity        int       `json:"quantity"`
	ProductID       uuid.UUID `json:"product_id"`
	ProductName     string    `json:"product_name"`
	ProductPrice    int       `json:"product_price"`
	CartDescription string    `json:"cart_description"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type CartItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	UserID      uuid.UUID `json:"user_id"`
	ProductName string    `json:"product_name"`
	Price       int       `json:"price"`
	Quantity    int       `json:"quantity"`
	Description string    `json:"description"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty"`
}

type CartRequest struct {
	ProductID   uuid.UUID `json:"product_id" validate:"required"`
	Quantity    int       `json:"quantity" validate:"required,min=1"`
	Description string    `json:"description"`
}

type UpdateCartRequest struct {
	Quantity    int    `json:"quantity" validate:"required"`
	Description string `json:"description"`
}
