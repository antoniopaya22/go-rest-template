package models

import (
	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model
	Username  string `gorm:"type:varchar(100);unique;not null" json:"username"`
	Lastname  string `gorm:"type:varchar(100);not null" json:"lastname"`
	Firstname string `gorm:"type:varchar(100);not null" json:"firstname"`
	Hash      string `gorm:"type:varchar(255);not null" json:"hash"`
}

type UserData struct {
	TotalData    int64
	FilteredData int64
	Data         []User
}