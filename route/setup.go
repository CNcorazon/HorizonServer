package route

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func InitRoute() *gin.Engine {
	fmt.Println(
		"213")
	r := gin.Default()
	blockRoute(r)
	ShardRoute(r)
	return r
}
