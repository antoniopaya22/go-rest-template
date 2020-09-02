package persistence

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/db"
	models "github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/models/users"
	"strconv"
)

// UserDAO persists user data in database
type UserDAO struct{}

// NewUserDAO creates a new UserDAO
func NewUserDAO() *UserDAO {
	return &UserDAO{}
}

func (dao *UserDAO) Get(id string) (*models.User, error) {
	var user models.User
	var userRole models.UserRole
	where := models.User{}
	where.ID, _ = strconv.ParseUint(id, 10, 64)
	_, err := First(&where, &user)
	if err != nil {
		return nil, err
	}
	err = db.GetDB().Model(&user).Association("Role").Find(&userRole).Error
	user.Role = userRole
	return &user, err
}

func (dao *UserDAO) GetByUsername(username string) (*models.User, error) {
	var user models.User
	var userRole models.UserRole
	where := models.User{}
	where.Username = username
	_, err := First(&where, &user)
	if err != nil {
		return nil, err
	}
	err = db.GetDB().Model(&user).Association("Role").Find(&userRole).Error
	user.Role = userRole
	return &user, err
}

func (dao *UserDAO) All() (*[]models.User, error) {
	var users []models.User
	err := Find(&models.User{}, &users, "id asc")
	for i := range users {
		var userRole models.UserRole
		err = db.GetDB().Model(&users[i]).Association("Role").Find(&userRole).Error
		users[i].Role = userRole
	}
	return &users, err
}

func (dao *UserDAO) Query(username string, firstname string, lastname string) (*[]models.User, error) {
	var users []models.User
	err := Find(&models.User{
		Username:  username,
		Lastname:  lastname,
		Firstname: firstname,
	}, &users, "id asc")
	for i := range users {
		var userRole models.UserRole
		err = db.GetDB().Model(&users[i]).Association("Role").Find(&userRole).Error
		users[i].Role = userRole
	}
	return &users, err
}

func (dao *UserDAO) Add(user *models.User) error {
	err := Create(&user)
	err = Save(&user)
	return err
}

func (dao *UserDAO) Update(user *models.User) error {
	var userRole models.UserRole
	_, err := First(models.UserRole{UserID: user.ID}, &userRole)
	userRole.RoleName = user.Role.RoleName
	err = Save(&userRole)
	err = db.GetDB().Omit("Role").Save(&user).Error
	user.Role = userRole
	return err
}

func (dao *UserDAO) Delete(user *models.User) error {
	err := db.GetDB().Unscoped().Delete(models.UserRole{UserID: user.ID}).Error
	err = db.GetDB().Unscoped().Delete(&user).Error
	return err
}
