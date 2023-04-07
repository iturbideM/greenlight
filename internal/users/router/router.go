package router

import "github.com/gin-gonic/gin"

type UserHandler interface {
	Register(c *gin.Context)
	ActivateUser(c *gin.Context)
}

type TokenHandler interface {
	CreateAuthenticationToken(c *gin.Context)
}

func InitRouter(engine *gin.RouterGroup, userhandler UserHandler, tokenHandler TokenHandler) {
	users := engine.Group("/users")
	{
		users.POST("", userhandler.Register)
		users.PUT("/activated", userhandler.ActivateUser)
	}

	token := engine.Group("/tokens")
	{
		token.POST("/authentication", tokenHandler.CreateAuthenticationToken)
	}
}
