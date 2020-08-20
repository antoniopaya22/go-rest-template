package daos

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/configs"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/utils"
	"github.com/gin-gonic/gin"
)

// UserDAO persists user data in database
type UserDAO struct{}


// NewUserDAO creates a new UserDAO
func NewUserDAO() *UserDAO {
	return &UserDAO{}
}

func (dao *UserDAO) Get(id string) (*models.User, error) {
	var user models.User
	err := configs.GetDB().Where("id = ?", id).
		First(&user).
		Error
	return &user, err
}

func (dao *UserDAO) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := configs.GetDB().Where("username = ?", username).
		First(&user).
		Error
	return &user, err
}

func (dao *UserDAO) All() (*[]models.User, error){
	var users []models.User
	err := configs.GetDB().Find(&users).Error
	return &users, err
}

func (dao *UserDAO) Add(user *models.User) error {
	return configs.GetDB().Create(&user).Error
}

func (dao *UserDAO) Update(user *models.User) error {
	return configs.GetDB().Save(&user).Error
}

func (dao *UserDAO) Delete(user *models.User) error {
	return configs.GetDB().Delete(&user).Error
}

func (dao *UserDAO) AllPaginated(c *gin.Context) (*models.UserData, error) {
	db := configs.GetDB()

	// Define and get sorting field
	sort := c.DefaultQuery("Sort", "ID")
	// Define and get sorting order field
	order := c.DefaultQuery("Order", "DESC")
	// Define and get offset for pagination
	offset := c.DefaultQuery("Offset", "0")
	// Define and get limit for pagination
	limit := c.DefaultQuery("Limit", "25")

	table := "users"
	query := db.Select(table + ".*")
	query = query.Offset(utils.Offset(offset))
	query = query.Limit(utils.Limit(limit))
	query = query.Order(utils.SortOrder(table, sort, order))

	var users []models.User
	err := query.Find(&users).Error

	var data models.UserData
	query = query.Offset(0)
	query.Table(table).Count(&data.FilteredData)

	// Count total table
	db.Table(table).Count(&data.TotalData)
	data.Data = users

	return &data, err
}