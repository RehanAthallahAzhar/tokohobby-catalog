package models

import (
	"time"

	"gorm.io/gorm"
)

type Session struct {
	gorm.Model
	Token    string
	Username string `gorm:"type:varchar(100);unique_index"`
	Expiry   time.Time
}
