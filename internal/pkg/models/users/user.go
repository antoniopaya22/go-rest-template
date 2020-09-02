package users

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/models"
	"time"
)

type User struct {
	models.Model
	Username  string   `gorm:"column:username;not null;" json:"username"`
	Firstname string   `gorm:"column:firstname;not null;" json:"firstname"`
	Lastname  string   `gorm:"column:lastname;not null;" json:"lastname"`
	Hash      string   `gorm:"column:hash;not null;" json:"hash"`
	Role      UserRole `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (m *User) BeforeCreate() error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

func (m *User) BeforeUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
