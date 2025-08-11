package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Price       int    `json:"price" validate:"required,gt=0"` // gt=0 berarti harus lebih besar dari 0
	Stock       int    `json:"stock" validate:"required,gte=0"`
	Discount    int    `json:"discount" validate:"gte=0,lte=100"` // gt=0,lte=100 (diskon 0-100%)
	Type        string `json:"type" validate:"required,alpha"`    // alpha berarti hanya huruf
	Description string `json:"description"`
}
type ProductResponse struct {
	ID          uuid.UUID `json:"id"`
	SellerID    uuid.UUID `json:"seller_id"`
	SellerName  string    `json:"seller_name"`
	Name        string    `json:"name"`
	Price       int       `json:"price"`
	Stock       int       `json:"stock"`
	Discount    int       `json:"discount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

type ProductWithSeller struct {
	ID          uuid.UUID `json:"id"`
	SellerID    uuid.UUID `json:"seller_id"`
	SellerName  string    `json:"seller_name"`
	Name        string    `json:"name"`
	Price       int       `json:"price"`
	Stock       int       `json:"stock"`
	Discount    int       `json:"discount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
