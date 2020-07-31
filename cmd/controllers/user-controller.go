package controllers

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/repository"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/crypto"
	"github.com/gin-gonic/gin"
	"net/http"
)

type UserInput struct {
	Username  string `json:"username" binding:"required"`
	Lastname  string `json:"lastname" binding:"required"`
	Firstname string `json:"firstname" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

func GetUserById(c *gin.Context) {
	id := c.Params.ByName("id")
	var user models.User
	if err := repository.FindUserById(&user, id); err != nil {
		c.JSON(http.StatusNotFound, "User not found")
		return
	}
	c.JSON(http.StatusOK, user)
}

func GetUsersPaginated(c *gin.Context) {
	var users []models.User
	var data repository.Data
	if err := repository.AllPaginatedUsers(&users, &data, c); err != nil {
		c.JSON(http.StatusNotFound, "Users not found")
		return
	}
	// Set Data result
	data.Data = users
	c.JSON(200, data)
}

func GetUsers(c *gin.Context)  {
	var users []models.User
	if err := repository.AllUsers(&users); err != nil {
		c.JSON(http.StatusNotFound, "Users not found")
		return
	}
	c.JSON(http.StatusOK, users)
}

func CreateUser(c *gin.Context) {
	var userInput UserInput
	c.BindJSON(&userInput)
	user := models.User{
		Username:  userInput.Username,
		Firstname: userInput.Firstname,
		Lastname:  userInput.Lastname,
		Hash:      crypto.HashAndSalt([]byte(userInput.Password)),
	}
	if err := repository.AddUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, user)
}

func UpdateUser(c *gin.Context) {
	id := c.Params.ByName("id")
	var userInput UserInput
	c.BindJSON(&userInput)
	var user models.User
	if err := repository.FindUserById(&user, id); err != nil {
		c.JSON(http.StatusNotFound, "User not found")
		return
	}
	user.Username = userInput.Username
	user.Lastname = userInput.Lastname
	user.Firstname = userInput.Firstname
	user.Hash = crypto.HashAndSalt([]byte(userInput.Password))
	if err := repository.UpdateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	id := c.Params.ByName("id")
	var user models.User
	if err := repository.FindUserById(&user, id); err != nil {
		c.JSON(http.StatusNotFound, "User not found")
		return
	}
	if err := repository.DeleteUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusNoContent, "")
}