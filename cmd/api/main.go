package main

import (
	_ "github.com/antonioalfa22/GoGin-API-REST-Template/docs"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/api"
)

// @title Go Gin Rest API
// @version 1.0
// @description API REST in Golang with Gin Framework
// @termsOfService http://swagger.io/terms/

// @contact.name Antonio Paya Gonzalez
// @contact.email antonioalfa22@gmail.com

// @license.name MIT
// @license.url https://github.com/antonioalfa22/GoGin-API-REST-Template/blob/master/LICENSE

// @BasePath /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main()  {
	api.Run("")
}