package controllers

import (
	"errors"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/daos"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/services"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/utils"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type UserInput struct {
	Username  string `json:"username" binding:"required"`
	Lastname  string `json:"lastname" binding:"required"`
	Firstname string `json:"firstname" binding:"required"`
	Password  string `json:"password" binding:"required"`
}


// GetUserById godoc
// @Summary Retrieves user based on given ID
// @Produce json
// @Param id path integer true "User ID"
// @Success 200 {object} models.User
// @Router /api/users/{id} [get]
// @Security Authorization Token
func GetUserById(c *gin.Context) {
	s := services.NewUserService(daos.NewUserDAO())
	id := c.Params.ByName("id")
	if user, err := s.Get(id); err != nil {
		utils.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, user)
	}
}

// GetUserById godoc
// @Summary Retrieves all users
// @Produce json
// @Success 200 {object} [] models.User
// @Router /api/users [get]
// @Security Authorization Token
func GetUsersPaginated(c *gin.Context) {
	s := services.NewUserService(daos.NewUserDAO())
	if users, err := s.AllPaginated(c); err != nil {
		utils.NewError(c, http.StatusNotFound, errors.New("users not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, users)
	}
}

func CreateUser(c *gin.Context) {
	s := services.NewUserService(daos.NewUserDAO())
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	user := models.User{
		Username:  userInput.Username,
		Firstname: userInput.Firstname,
		Lastname:  userInput.Lastname,
		Hash:      utils.HashAndSalt([]byte(userInput.Password)),
	}
	if err := s.Add(&user); err != nil {
		utils.NewError(c, http.StatusNotFound, err)
		log.Println(err)
	} else {
		c.JSON(http.StatusCreated, user)
	}
}

func UpdateUser(c *gin.Context) {
	s := services.NewUserService(daos.NewUserDAO())
	id := c.Params.ByName("id")
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	if user, err := s.Get(id); err != nil {
		utils.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		user.Username = userInput.Username
		user.Lastname = userInput.Lastname
		user.Firstname = userInput.Firstname
		user.Hash = utils.HashAndSalt([]byte(userInput.Password))
		if err := s.Update(user); err != nil {
			utils.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusOK, user)
		}
	}
}

func DeleteUser(c *gin.Context) {
	s := services.NewUserService(daos.NewUserDAO())
	id := c.Params.ByName("id")
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	if user, err := s.Get(id); err != nil {
		utils.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		if err := s.Delete(user); err != nil {
			utils.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusNoContent, "")
		}
	}
}