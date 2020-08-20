package main

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/api/routes"
	"github.com/antonioalfa22/GoGin-API-REST-Template/configs"
	_ "github.com/antonioalfa22/GoGin-API-REST-Template/docs"
	"github.com/gin-gonic/gin"
)

func init()  {
	configs.Setup()
	configs.SetupDB()
	gin.SetMode(configs.GetConfig().Server.Mode)
}

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
	config := configs.GetConfig()
	r := routes.Setup()
	fmt.Println("Go API REST Running on port "+config.Server.Port)
	fmt.Println("==================>")
	_ = r.Run("127.0.0.1:" + config.Server.Port)
}