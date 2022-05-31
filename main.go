package main

import (
	"io"
	"log"
	"os"
	"server/route"
	"server/structure"
	"time"

	"github.com/gin-gonic/gin"
)

// func GetLocalIP() []string {
// 	var ipStr []string
// 	netInterfaces, err := net.Interfaces()
// 	if err != nil {
// 		fmt.Println("net.Interfaces error:", err.Error())
// 		return ipStr
// 	}

// 	for i := 0; i < len(netInterfaces); i++ {
// 		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
// 			addrs, _ := netInterfaces[i].Addrs()
// 			for _, address := range addrs {
// 				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
// 					//获取IPv6
// 					/*if ipnet.IP.To16() != nil {
// 						fmt.Println(ipnet.IP.String())
// 						ipStr = append(ipStr, ipnet.IP.String())

// 					}*/
// 					//获取IPv4
// 					if ipnet.IP.To4() != nil {
// 						fmt.Println(ipnet.IP.String())
// 						ipStr = append(ipStr, ipnet.IP.String())

// 					}
// 				}
// 			}
// 		}
// 	}
// 	return ipStr

// }
func main() {
	// //设置gin框架的日志
	gin.DisableConsoleColor()
	// // 创建日志文件并设置为 gin.DefaultWriter
	f, _ := os.Create("gin.log")
	gin.DefaultWriter = io.MultiWriter(f)

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	go func() {
		for {
			time.Sleep(5 * time.Second)
			structure.Source.ChainShard[uint(0)].AccountState.LogState(structure.Source.ChainShard[uint(0)].GetHeight())
			for i := 1; i <= structure.ShardNum; i++ {
				structure.Source.PoolMap[uint(0)][uint(i)].LogState()
				structure.Source.PoolMap[uint(1)][uint(i)].LogState()
				structure.Source.PoolMap[uint(2)][uint(i)].LogState()
			}
		}
	}()
	r := route.InitRoute()
	r.Run()
}
