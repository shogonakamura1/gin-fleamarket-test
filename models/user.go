package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email    string `gorm:"not null;unique"`
	Password string `gorm:"not null"`
	Role     string `gorm:"not null;default:'user'"`
	Items    []Item `gorm:"constraint:OnDelete:CASCADE;"`
}
