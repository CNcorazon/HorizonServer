package route

import (
	"server/controller"

	"github.com/gin-gonic/gin"
)

func ForwardRoute(r *gin.Engine) {
	group := r.Group("/Forward")
	{
		group.GET("/wsRequest/:ip", controller.ServerRegisterCommunication)
		// group.POST("/register", controller.ForwardRegisterCommunication)
	}
}
