package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SellerID    uuid.UUID `gorm:"type:uuid" json:"seller_id"`
	Name        string    `gorm:"type:varchar(100)" json:"name"`
	Price       int       `json:"price" `
	Stock       int       `json:"stock"`
	Discount    int       `json:"discount"`
	Type        string    `json:"type"`
	Description string    `gorm:"type:text" json:"description"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

func (c *Product) TableName() string {
	return "product"
}
