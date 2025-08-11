package registry

import (
	"gorm.io/gorm"

	config "github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/config"
)

type Registry struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewRegistry(cfg *config.Config, db *gorm.DB) *Registry {
	return &Registry{
		cfg: cfg,
		db:  db,
	}
}
