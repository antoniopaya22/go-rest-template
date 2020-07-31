package controllers

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/repository"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/crypto"
	"github.com/gin-gonic/gin"
	"net/http"
)


type LoginInput struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var loginInput LoginInput
	c.BindJSON(&loginInput)
	var user models.User
	if err := repository.FindUserByUsername(&user, loginInput.Username); err != nil {
		c.JSON(http.StatusNotFound, "User not found")
		return
	}

	if ! crypto.ComparePasswords(user.Hash, []byte(loginInput.Password)){
		c.JSON(http.StatusForbidden, "User and password not match")
		return
	}
	token, _ := crypto.CreateToken(user.Username)
	c.JSON(http.StatusOK, token)
}
