package models

import "time"

type Order struct {
	ID          string      `gorm:"primaryKey" json:"id"`
	UserID      string      `gorm:"type:varchar(255);not null" json:"user_id"`
	TotalAmount int         `gorm:"type:int;not null" json:"total_amount"`
	Status      string      `gorm:"type:varchar(50);not null" json:"status"`
	OrderDate   time.Time   `gorm:"not null" json:"order_date"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	OrderItems  []OrderItem `gorm:"foreignKey:OrderID" json:"order_items"`
}

type OrderItem struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	OrderID   string    `gorm:"not null" json:"order_id"`
	ProductID string    `gorm:"not null" json:"product_id"`
	Quantity  int       `gorm:"type:integer;not null" json:"quantity"`
	Price     int       `gorm:"type:int;not null" json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
