package routes

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/api/controllers"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/api/middlewares"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

func Setup() *gin.Engine {
	r := gin.New()
	// Middlewares
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middlewares.CORS())

	// Users Routes
	users := r.Group("/api/users")
	users.Use(middlewares.AuthRequired())
	{
		users.GET("/", controllers.GetUsersPaginated)
		users.GET("/:id", controllers.GetUserById)
		users.POST("/", controllers.CreateUser)
		users.PUT("/:id", controllers.UpdateUser)
		users.DELETE("/:id", controllers.DeleteUser)
	}

	// Login Routes
	login := r.Group("/api/login")
	{
		login.POST("/", controllers.Login)
	}

	// Docs Routes
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
