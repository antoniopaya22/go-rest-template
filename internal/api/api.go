package api

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/api/middlewares"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/api/router"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/config"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/db"
	"github.com/gin-gonic/gin"
)

func setConfiguration(configPath string) {
	config.Setup(configPath)
	db.SetupDB()
	gin.SetMode(config.GetConfig().Server.Mode)
}

func initWeb() *gin.Engine {
	app := gin.New()

	// Middlewares
	app.Use(gin.Logger())
	app.Use(gin.Recovery())
	app.Use(middlewares.CORS())
	app.NoRoute(middlewares.NoRouteHandler())

	// Routes
	router.RegisterAuthRoutes(app)
	router.RegisterDocsRoutes(app)
	router.RegisterUserRoutes(app)

	return app
}

func Run(configPath string) {
	if configPath == "" {
		configPath = "./cmd/api/config.yml"
	}
	setConfiguration(configPath)
	conf := config.GetConfig()
	web := initWeb()
	fmt.Println("Go API REST Running on port " + conf.Server.Port)
	fmt.Println("==================>")
	_ = web.Run("127.0.0.1:" + conf.Server.Port)
}
