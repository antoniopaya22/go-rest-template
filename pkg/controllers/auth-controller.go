package controllers

import (
	"errors"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/daos"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/services"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/utils"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)


type LoginInput struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var loginInput LoginInput
	_ = c.BindJSON(&loginInput)
	s := services.NewUserService(daos.NewUserDAO())
	if user, err := s.GetByUsername(loginInput.Username); err != nil {
		utils.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		if ! utils.ComparePasswords(user.Hash, []byte(loginInput.Password)){
			utils.NewError(c, http.StatusForbidden, errors.New("user and password not match"))
			return
		}
		token, _ := utils.CreateToken(user.Username)
		c.JSON(http.StatusOK, token)
	}
}
