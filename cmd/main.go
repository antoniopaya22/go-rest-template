package main

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/routes"
	_ "github.com/antonioalfa22/GoGin-API-REST-Template/docs"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/config"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/database"
	"github.com/gin-gonic/gin"
)

func init()  {
	config.Setup()
	database.Setup()
	gin.SetMode(config.GetConfig().Server.Mode)
}

func main()  {
	config := config.GetConfig()
	r := routes.Setup()
	fmt.Println("Go API REST Running on port "+config.Server.Port)
	fmt.Println("==================>")
	r.Run("127.0.0.1:" + config.Server.Port)
}