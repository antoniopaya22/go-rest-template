package api

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/api/middlewares"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/api/router"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/config"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/db"
	"github.com/gin-gonic/gin"
	"io"
	"os"
)

func setConfiguration(configPath string) {
	config.Setup(configPath)
	db.SetupDB()
	gin.SetMode(config.GetConfig().Server.Mode)
}

func initWeb() *gin.Engine {
	app := gin.New()

	// Logging to a file.
	f, _ := os.Create("log/api.log")
	gin.DisableConsoleColor()
	gin.DefaultWriter = io.MultiWriter(f)

	// Middlewares
	app.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - - [%s] \"%s %s %s %d %s \" \" %s\" \" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
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
