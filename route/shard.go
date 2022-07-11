package route

import (
	"server/controller"

	"github.com/gin-gonic/gin"
)

func ShardRoute(r *gin.Engine) {
	group := r.Group("/shard")
	{
		group.GET("/shardNum", controller.AssignShardNum)
		group.GET("/height", controller.GetHeight)
		group.GET("/register/:shardnum/:random", controller.RegisterCommunication)
		// group.GET("/phase/:shardnum", controller.GetPhase)
		group.POST("/block", controller.MultiCastBlock)
		group.POST("/vote", controller.SendVote)
	}
}
