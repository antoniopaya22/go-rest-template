package api

import (
	"fmt"

	"github.com/antonioalfa22/go-rest-template/internal/api/router"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/config"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/db"
	"github.com/gin-gonic/gin"
)

func setConfiguration(configPath string) {
	config.Setup(configPath)
	db.SetupDB()
	gin.SetMode(config.GetConfig().Server.Mode)
}

func Run(configPath string) {
	if configPath == "" {
		configPath = "data/config.yml"
	}
	setConfiguration(configPath)
	conf := config.GetConfig()
	web := router.Setup()
	fmt.Println("Go API REST Running on port " + conf.Server.Port)
	fmt.Println("==================>")
	_ = web.Run(":" + conf.Server.Port)
}
