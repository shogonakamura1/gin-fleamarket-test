package models

import "gorm.io/gorm"

type BlacklistedToken struct {
	gorm.Model
	Token     string `gorm:"not null;unique;index"`
	ExpiresAt int64  `gorm:"not null;index"`
}
