package router

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/api/controllers"
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/api/middlewares"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

func RegisterUserRoutes(r *gin.Engine)  {
	// Users Routes
	users := r.Group("/api/users")
	users.Use(middlewares.AuthRequired())
	{
		users.GET("/", controllers.GetUsers)
		users.GET("/:id", controllers.GetUserById)
		users.POST("/", controllers.CreateUser)
		users.PUT("/:id", controllers.UpdateUser)
		users.DELETE("/:id", controllers.DeleteUser)
	}
}

func RegisterAuthRoutes(r *gin.Engine)  {
	// Login Routes
	login := r.Group("/api/login")
	{
		login.POST("/", controllers.Login)
	}
}

func RegisterDocsRoutes(r *gin.Engine)  {
	// Docs Routes
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
