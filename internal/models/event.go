package models

import "time"

// OrderCreatedEvent adalah struktur data untuk event yang dipublikasikan setelah pesanan dibuat.
type OrderCreatedEvent struct {
	OrderID     string         `json:"order_id"`
	UserID      string         `json:"user_id"`
	TotalAmount int            `json:"total_amount"`
	OrderDate   time.Time      `json:"order_date"`
	ProductIDs  []string       `json:"product_ids"` // IDs produk yang dibeli
	Quantities  map[string]int `json:"quantities"`  // Kuantitas untuk setiap produk ID
	// Anda bisa menambahkan detail lain yang relevan seperti alamat pengiriman, dll.
}
