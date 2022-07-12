package controller

import (
	"fmt"
	"log"
	"net/http"
	"server/logger"
	"server/structure"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func ServerRegisterCommunication(c *gin.Context) {
	ip := c.Param("ip")
	fmt.Println("im here")
	//将http请求升级成为WebSocket请求
	upGrader := websocket.Upgrader{
		// cross origin domain
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		// 处理 Sec-WebSocket-Protocol Header
		Subprotocols: []string{c.GetHeader("Sec-WebSocket-Protocol")},
	}

	conn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket connect error: %s", c.Param("channel"))
		return
	}

	structure.Source.Lock.Lock()
	defer structure.Source.Lock.Unlock()

	// if structure.Source.NodeNum[uint(shardnum)] == structure.CLIENT_MAX {
	// 	conn.Close()
	// 	logger.ShardLogger.Printf("分片%v已经选到了足够多的移动节点进行执行", shardnum)
	// 	return
	// }

	//尝试添加节点
	server := &structure.Server{
		Ip:     ip,
		Socket: conn,
	}

	// logger.AnalysisLogger.Printf("consensus_map:%v", structure.Source.Consensus_CommunicationMap)
	// logger.AnalysisLogger.Printf("validation_map:%v", structure.Source.Validation_CommunicationMap)

	Server_Map := structure.Source.Server_CommunicationMap

	if Server_Map == nil {
		Server_Map = make(map[string]*structure.Server)
	}

	Server_Map[ip] = server

	logger.ShardLogger.Printf("连接到一个新的服务器%v,当前服务器数为%v", ip, len(Server_Map))
}

// func ForwardRegisterCommunication(c *gin.Context) {
// 	var data model.ClientForwardRequest
// 	//判断请求的结构体是否符合定义
// 	if err := c.ShouldBindJSON(&data); err != nil {
// 		// gin.H封装了生成json数据的工具
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	structure.Source.Lock.Lock()
// 	defer structure.Source.Lock.Unlock()

// 	//尝试添加节点
// 	client := &structure.Client{
// 		Id:    data.Id,
// 		Shard: data.Shard,
// 		// Socket: ,
// 		Random: data.Random,
// 	}
// 	shardnum := data.Shard
// 	// logger.AnalysisLogger.Printf("consensus_map:%v", structure.Source.Consensus_CommunicationMap)
// 	// logger.AnalysisLogger.Printf("validation_map:%v", structure.Source.Validation_CommunicationMap)

// 	Consensus_Map := structure.Source.Consensus_CommunicationMap
// 	Validation_Map := structure.Source.Validation_CommunicationMap

// 	if Consensus_Map[uint(shardnum)] == nil {
// 		Consensus_Map[uint(shardnum)] = make(map[string]*structure.Client)
// 	}

// 	Consensus_Map[uint(shardnum)][client.Id] = client
// 	logger.ShardLogger.Printf("分片%v添加一个新的移动节点,当前分片的移动节点数为%v", shardnum, len(Consensus_Map[uint(shardnum)]))

// 	//如果该执行分片达到了足够多的移动节点数目,从该执行分片中选出胜者加入共识分片，shard[0]
// 	if len(Consensus_Map[uint(shardnum)]) == structure.CLIENT_MAX {
// 		// structure.Source.Phase[uint(shardnum)] = 2 //切换到第二阶段：生成委员会阶段
// 		logger.ShardLogger.Printf("执行分片%v切换到了生成委员会阶段", shardnum)
// 		//根据Client提交的Random筛选出本执行分片中的胜利者，进入委员会
// 		Win := client.Id
// 		WinRandom := client.Random
// 		WinClient := client
// 		for key, value := range Consensus_Map[uint(shardnum)] {
// 			if value.Random < WinRandom {
// 				Win = key
// 				WinRandom = value.Random //选出Random最小的作为胜利者
// 				WinClient = value
// 			}
// 		}
// 		//胜利者进入共识分片，先从执行分片中删除
// 		delete(Consensus_Map[uint(shardnum)], Win)

// 		if Consensus_Map[uint(0)] == nil {
// 			Consensus_Map[uint(0)] = make(map[string]*structure.Client, structure.ShardNum)
// 		}
// 		//记录进分片0，即委员会
// 		Consensus_Map[uint(0)][Win] = WinClient
// 		// logger.AnalysisLogger.Printf("分片%v填满节点后consensus_map中的节点有:%v", shardnum, structure.Source.Consensus_CommunicationMap)
// 		// logger.AnalysisLogger.Printf("分片%v填满节点后validation_map中的节点有:%v", shardnum, structure.Source.Validation_CommunicationMap)

// 		if Validation_Map[uint(shardnum)] == nil {
// 			Validation_Map[uint(shardnum)] = Consensus_Map[uint(shardnum)]
// 		}

// 		//若所有执行分片都选出了胜者，则在委员会中选出最终胜者
// 		if len(Consensus_Map[uint(0)]) == structure.ShardNum {
// 			logger.ShardLogger.Printf("所有执行分片都选出了胜者，开始进行共识")

// 			FinalWin := WinClient.Id
// 			FinalRandom := WinClient.Random
// 			var idlist []string

// 			for key, value := range Consensus_Map[uint(0)] {
// 				if value.Random < FinalRandom {
// 					FinalWin = key
// 					FinalRandom = value.Random //选出Random最小的作为胜利者
// 				}
// 				idlist = append(idlist, key)
// 			}
// 			structure.Source.Winner[uint(0)] = FinalWin

// 			//通知共识节点胜出者的身份
// 			for key, value := range Consensus_Map[uint(0)] {
// 				if value.Socket == nil {
// 					continue
// 				}
// 				if key == FinalWin {
// 					message := model.MessageIsWin{
// 						IsWin:       true,
// 						IsConsensus: true,
// 						WinID:       FinalWin,
// 						PersonalID:  FinalWin,
// 						IdList:      idlist,
// 					}
// 					payload, err := json.Marshal(message)
// 					if err != nil {
// 						log.Println(err)
// 						return
// 					}
// 					metamessage := model.MessageMetaData{
// 						MessageType: 1,
// 						Message:     payload,
// 					}
// 					value.Socket.WriteJSON(metamessage)
// 				} else {
// 					message := model.MessageIsWin{
// 						IsWin:       false,
// 						IsConsensus: true,
// 						WinID:       FinalWin,
// 						PersonalID:  key,
// 						IdList:      idlist,
// 					}
// 					payload, err := json.Marshal(message)
// 					if err != nil {
// 						log.Println(err)
// 						return
// 					}
// 					metamessage := model.MessageMetaData{
// 						MessageType: 1,
// 						Message:     payload,
// 					}
// 					value.Socket.WriteJSON(metamessage)
// 				}
// 			}
// 			for i := 1; i <= structure.ShardNum; i++ {
// 				//通知执行节点胜出者的身份
// 				for key, value := range Consensus_Map[uint(i)] {
// 					if value.Socket == nil {
// 						continue
// 					}
// 					message := model.MessageIsWin{
// 						IsWin:       false,
// 						IsConsensus: false,
// 						WinID:       FinalWin,
// 						PersonalID:  key,
// 						IdList:      idlist,
// 					}
// 					payload, err := json.Marshal(message)
// 					if err != nil {
// 						log.Println(err)
// 						return
// 					}
// 					metamessage := model.MessageMetaData{
// 						MessageType: 1,
// 						Message:     payload,
// 					}
// 					value.Socket.WriteJSON(metamessage)
// 				}
// 			}
// 		}
// 	}
// 	res := model.ClientForwardResponse{
// 		Msg: "register success",
// 	}
// 	c.JSON(200, res)
// }
