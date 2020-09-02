package controllers

import (
	"errors"
	models "github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/models/users"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/services"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/crypto"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/http-err"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type UserInput struct {
	Username  string `json:"username" binding:"required"`
	Lastname  string `json:"lastname" binding:"required"`
	Firstname string `json:"firstname" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Role      string `json:"role" binding:"required"`
}

type QueryUserParams struct {
	Username  string `form:"username"`
	Lastname  string `form:"lastname"`
	Firstname string `form:"firstname"`
}

// GetUserById godoc
// @Summary Retrieves user based on given ID
// @Produce json
// @Param id path integer true "User ID"
// @Success 200 {object} models.User
// @Router /api/users/{id} [get]
// @Security Authorization Token
func GetUserById(c *gin.Context) {
	s := services.GetUserService()
	id := c.Params.ByName("id")
	if user, err := s.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, user)
	}
}

// GetUsers godoc
// @Summary Retrieves users based on query
// @Produce json
// @Param id path integer true "User ID"
// @Success 200 {object} models.User
// @Router /api/users/{id} [get]
// @Security Authorization Token
func GetUsers(c *gin.Context) {
	s := services.GetUserService()
	var userQuery QueryUserParams
	_ = c.Bind(&userQuery)
	if users, err := s.Query(userQuery.Username, userQuery.Firstname, userQuery.Lastname); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("users not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, users)
	}
}

func CreateUser(c *gin.Context) {
	s := services.GetUserService()
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	user := models.User{
		Username:  userInput.Username,
		Firstname: userInput.Firstname,
		Lastname:  userInput.Lastname,
		Hash:      crypto.HashAndSalt([]byte(userInput.Password)),
		Role:      models.UserRole{RoleName: userInput.Role},
	}
	if err := s.Add(&user); err != nil {
		http_err.NewError(c, http.StatusNotFound, err)
		log.Println(err)
	} else {
		c.JSON(http.StatusCreated, user)
	}
}

func UpdateUser(c *gin.Context) {
	s := services.GetUserService()
	id := c.Params.ByName("id")
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	if user, err := s.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		user.Username = userInput.Username
		user.Lastname = userInput.Lastname
		user.Firstname = userInput.Firstname
		user.Hash = crypto.HashAndSalt([]byte(userInput.Password))
		user.Role = models.UserRole{RoleName: userInput.Role}
		if err := s.Update(user); err != nil {
			http_err.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusOK, user)
		}
	}
}

func DeleteUser(c *gin.Context) {
	s := services.GetUserService()
	id := c.Params.ByName("id")
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	if user, err := s.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		if err := s.Delete(user); err != nil {
			http_err.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusNoContent, "")
		}
	}
}
