package controllers

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/repository"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/services"
	"github.com/gin-gonic/gin"
	"net/http"
)


type LoginInput struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var loginInput LoginInput
	_ = c.BindJSON(&loginInput)
	var user models.User
	if err := repository.FindUserByUsername(&user, loginInput.Username); err != nil {
		c.JSON(http.StatusNotFound, "User not found")
		return
	}

	if ! services.ComparePasswords(user.Hash, []byte(loginInput.Password)){
		c.JSON(http.StatusForbidden, "User and password not match")
		return
	}
	token, _ := services.CreateToken(user.Username)
	c.JSON(http.StatusOK, token)
}
