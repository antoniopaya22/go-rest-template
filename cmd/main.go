package main

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/routes"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/config"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/database"
)

func init()  {
	config.Setup()
	database.Setup()
}

func main()  {
	config := config.GetConfig()
	r := routes.Setup()
	fmt.Println("Go API REST Running on port "+config.Server.Port)
	fmt.Println("==================>")
	r.Run("127.0.0.1:" + config.Server.Port)
}