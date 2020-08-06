package main

import (
	"fmt"
	_ "github.com/antonioalfa22/GoGin-API-REST-Template/docs"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/routes"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/services"
	"github.com/gin-gonic/gin"
)

func init()  {
	services.Setup()
	services.SetupDB()
	gin.SetMode(services.GetConfig().Server.Mode)
}

func main()  {
	config := services.GetConfig()
	r := routes.Setup()
	fmt.Println("Go API REST Running on port "+config.Server.Port)
	fmt.Println("==================>")
	_ = r.Run("127.0.0.1:" + config.Server.Port)
}