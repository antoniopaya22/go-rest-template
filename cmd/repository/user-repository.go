package repository

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/database"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

var db *gorm.DB

type Data struct {
	TotalData    int64
	FilteredData int64
	Data         []models.User
}


func AllUsers(users *[]models.User) error {
	db = database.GetDB()
	return db.Find(&users).Error
}

func FindUserById(user *models.User, id string) error {
	db = database.GetDB()
	return db.Find(&user, id).Error
}

func FindUserByUsername(user *models.User, username string) error {
	db = database.GetDB()
	return db.Where("username = ?", username).First(&user).Error
}

func AddUser(user *models.User) error {
	db = database.GetDB()
	return db.Create(&user).Error
}

func UpdateUser(user *models.User) error {
	db = database.GetDB()
	return db.Save(&user).Error
}

func DeleteUser(user *models.User) error {
	db = database.GetDB()
	return db.Delete(&user).Error
}

func AllPaginatedUsers(users *[]models.User, data *Data, c *gin.Context) error {
	db = database.GetDB()

	// Define and get sorting field
	sort := c.DefaultQuery("Sort", "ID")
	// Define and get sorting order field
	order := c.DefaultQuery("Order", "DESC")
	// Define and get offset for pagination
	offset := c.DefaultQuery("Offset", "0")
	// Define and get limit for pagination
	limit := c.DefaultQuery("Limit", "25")
	// Get search keyword for Search Scope
	search := c.DefaultQuery("Search", "")

	table := "users"
	query := db.Select(table + ".*")
	query = query.Offset(database.Offset(offset))
	query = query.Limit(database.Limit(limit))
	query = query.Order(database.SortOrder(table, sort, order))
	query = query.Scopes(database.Search(search))

	error := db.Find(&users).Error

	query = query.Offset(0)
	query.Table(table).Count(&data.FilteredData)

	// Count total table
	db.Table(table).Count(&data.TotalData)
	return error
}