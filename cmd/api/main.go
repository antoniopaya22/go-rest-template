package main

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/configs"
	_ "github.com/antonioalfa22/GoGin-API-REST-Template/docs"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/routes"
	"github.com/gin-gonic/gin"
)

func init()  {
	configs.Setup()
	configs.SetupDB()
	gin.SetMode(configs.GetConfig().Server.Mode)
}

func main()  {
	config := configs.GetConfig()
	r := routes.Setup()
	fmt.Println("Go API REST Running on port "+config.Server.Port)
	fmt.Println("==================>")
	_ = r.Run("127.0.0.1:" + config.Server.Port)
}