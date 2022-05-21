package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"server/route"

	"github.com/gin-gonic/gin"
)

func main() {
	// //设置gin框架的日志
	fmt.Println("1232131")
	gin.DisableConsoleColor()
	// // 创建日志文件并设置为 gin.DefaultWriter
	f, _ := os.Create("gin.log")
	gin.DefaultWriter = io.MultiWriter(f)

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	// go func() {
	// 	for {
	// 		time.Sleep(5 * time.Second)
	// 		structure.Source.ChainShard[uint(0)].AccountState.LogState(structure.Source.ChainShard[uint(0)].GetHeight())
	// 		for i := 1; i <= structure.ShardNum; i++ {
	// 			structure.Source.PoolMap[uint(0)][uint(i)].LogState()
	// 			structure.Source.PoolMap[uint(1)][uint(i)].LogState()
	// 			structure.Source.PoolMap[uint(2)][uint(i)].LogState()
	// 		}
	// 	}
	// }()
	r := route.InitRoute()
	r.Run()
}
