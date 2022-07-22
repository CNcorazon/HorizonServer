package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"server/logger"
	"server/model"
	"server/structure"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

const (
	WsRequest     = "/forward/wsRequest"
	ClientForward = "/forward/clientRegister "
)

//给发起请求的客户端分配分片的ID
func AssignShardNum(c *gin.Context) {
	structure.Source.Lock.Lock()
	defer structure.Source.Lock.Unlock()
	num := 0 //如果发送的时候为0说明执行分片已经有了足够多的节点

	for i := 1; i <= int(structure.Source.Shard); i++ {
		if structure.Source.NodeNum[uint(i)] == structure.CLIENT_MAX {
			continue
		} else {
			num = i
			structure.Source.NodeNum[uint(i)] += 1
			logger.AnalysisLogger.Printf("同步重分片各个分片的数量,此时各个分片的节点数量为%v", structure.Source.NodeNum)
			//直接转发给其他服务器
			//只考虑理想情况，固定数量的移动节点
			//会不会出现，服务器之间来不及同步的情况呢？
			for _, value := range structure.Source.Server_CommunicationMap {
				message := model.ReshardNodeNumRequest{
					Shard_id: uint(i),
					Nodenum:  structure.Source.NodeNum,
				}
				payload, err := json.Marshal(message)
				if err != nil {
					log.Println(err)
					return
				}
				metamessage := model.MessageMetaData{
					MessageType: 4,
					Message:     payload,
				}
				value.Lock.Lock()
				value.Socket.WriteJSON(metamessage)
				value.Lock.Unlock()
			}
			break
		}
	}

	logger.ShardLogger.Printf("该移动节点被分配到了%v分片", num)

	res := model.ShardNumResponse{
		ShardNum: uint(num),
	}
	c.JSON(200, res)
}

func RegisterCommunication(c *gin.Context) {
	shardnum, _ := strconv.Atoi(c.Param("shardnum"))
	random, _ := strconv.Atoi(c.Param("random"))

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
	client := &structure.Client{
		Id:     uuid.NewV4().String(),
		Shard:  uint(shardnum),
		Socket: conn,
		Random: random,
	}

	logger.AnalysisLogger.Printf("consensus_map:%v", structure.Source.Consensus_CommunicationMap)
	logger.AnalysisLogger.Printf("validation_map:%v", structure.Source.Validation_CommunicationMap)

	Consensus_Map := structure.Source.Consensus_CommunicationMap
	Validation_Map := structure.Source.Validation_CommunicationMap

	if Consensus_Map[uint(shardnum)] == nil {
		Consensus_Map[uint(shardnum)] = make(map[string]*structure.Client)
	}

	Consensus_Map[uint(shardnum)][client.Id] = client
	logger.ShardLogger.Printf("分片%v添加一个新的移动节点,当前分片的移动节点数为%v", shardnum, len(Consensus_Map[uint(shardnum)]))

	//将该节点信息转发给其他服务器
	message := model.ClientForwardRequest{
		Id:    client.Id,
		Shard: client.Shard,
		// Socket: client.Socket,
		Random: client.Random,
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
		return
	}
	metamessage := model.MessageMetaData{
		MessageType: 1,
		Message:     payload,
	}

	for _, value := range structure.Source.Server_CommunicationMap {
		value.Lock.Lock()
		value.Socket.WriteJSON(metamessage)
		value.Lock.Unlock()
	}

	// res1 := request.ClientRegister(structure.Server1, ClientForward, message)
	// res2 := request.ClientRegister(structure.Server2, ClientForward, message)
	// res3 := request.ClientRegister(structure.Server3, ClientForward, message)
	// fmt.Println(res1, res2, res3)
	// logger.AnalysisLogger.Printf("%v进入分片之后共识:%v", client.Id, Consensus_Map)
	//如果该执行分片达到了足够多的移动节点数目,从该执行分片中选出胜者加入共识分片，shard[0]
	if len(Consensus_Map[uint(shardnum)]) == structure.CLIENT_MAX {
		// structure.Source.Phase[uint(shardnum)] = 2 //切换到第二阶段：生成委员会阶段
		logger.ShardLogger.Printf("执行分片%v切换到了生成委员会阶段", shardnum)
		//根据Client提交的Random筛选出本执行分片中的胜利者，进入委员会
		Win := client.Id
		WinRandom := client.Random
		WinClient := client
		for key, value := range Consensus_Map[uint(shardnum)] {
			if value.Random < WinRandom {
				Win = key
				WinRandom = value.Random //选出Random最小的作为胜利者
				WinClient = value
			}
		}
		//胜利者进入共识分片，先从执行分片中删除
		delete(Consensus_Map[uint(shardnum)], Win)

		if Consensus_Map[uint(0)] == nil {
			Consensus_Map[uint(0)] = make(map[string]*structure.Client, structure.ShardNum)
		}
		//记录进分片0，即委员会
		Consensus_Map[uint(0)][Win] = WinClient
		logger.AnalysisLogger.Printf("分片%v填满节点后consensus_map中的节点有:%v", shardnum, structure.Source.Consensus_CommunicationMap)
		logger.AnalysisLogger.Printf("分片%v填满节点后validation_map中的节点有:%v", shardnum, structure.Source.Validation_CommunicationMap)

		if Validation_Map[uint(shardnum)] == nil {
			Validation_Map[uint(shardnum)] = Consensus_Map[uint(shardnum)]
		}

		//若所有执行分片都选出了胜者，则在委员会中选出最终胜者
		if len(Consensus_Map[uint(0)]) == structure.ShardNum {
			logger.ShardLogger.Printf("所有执行分片都选出了胜者，开始进行共识")

			FinalWin := WinClient.Id
			FinalRandom := WinClient.Random
			var idlist []string

			for key, value := range Consensus_Map[uint(0)] {
				if value.Random < FinalRandom {
					FinalWin = key
					FinalRandom = value.Random //选出Random最小的作为胜利者
				}
				idlist = append(idlist, key)
			}
			structure.Source.Winner[uint(0)] = FinalWin

			//通知共识节点胜出者的身份
			for key, value := range Consensus_Map[uint(0)] {
				if value.Socket == nil {
					continue
				}
				if key == FinalWin {
					message := model.MessageIsWin{
						IsWin:       true,
						IsConsensus: true,
						WinID:       FinalWin,
						PersonalID:  FinalWin,
						IdList:      idlist,
					}
					payload, err := json.Marshal(message)
					if err != nil {
						log.Println(err)
						return
					}
					metamessage := model.MessageMetaData{
						MessageType: 1,
						Message:     payload,
					}
					value.Socket.WriteJSON(metamessage)
				} else {
					message := model.MessageIsWin{
						IsWin:       false,
						IsConsensus: true,
						WinID:       FinalWin,
						PersonalID:  key,
						IdList:      idlist,
					}
					payload, err := json.Marshal(message)
					if err != nil {
						log.Println(err)
						return
					}
					metamessage := model.MessageMetaData{
						MessageType: 1,
						Message:     payload,
					}
					value.Socket.WriteJSON(metamessage)
				}
			}
			for i := 1; i <= structure.ShardNum; i++ {
				//通知执行节点胜出者的身份
				for key, value := range Consensus_Map[uint(i)] {
					if value.Socket == nil {
						continue
					}
					message := model.MessageIsWin{
						IsWin:       false,
						IsConsensus: false,
						WinID:       FinalWin,
						PersonalID:  key,
						IdList:      idlist,
					}
					payload, err := json.Marshal(message)
					if err != nil {
						log.Println(err)
						return
					}
					metamessage := model.MessageMetaData{
						MessageType: 1,
						Message:     payload,
					}
					value.Socket.WriteJSON(metamessage)
				}
			}
		}
	}
}

//共识分片内部的胜利者计算出区块之后，使用该函数向分片内部的节点转发计算出来得到的区块
func MultiCastBlock(c *gin.Context) {
	var data model.MultiCastBlockRequest
	//判断请求的结构体是否符合定义
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	structure.Source.Lock.Lock()
	defer structure.Source.Lock.Unlock()
	payload, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	metaMessage := model.MessageMetaData{
		MessageType: 2,
		Message:     payload,
	}

	// var CommunicationMap map[uint]map[string]*structure.Client
	// if len(structure.Source.CommunicationMap[uint(0)]) > len(structure.Source.CommunicationMap_temp[uint(0)]) {
	// 	CommunicationMap = structure.Source.CommunicationMap
	// } else {
	// 	CommunicationMap = structure.Source.CommunicationMap_temp
	// }

	//向连接在该服务器的委员会成员转发数据，注意不用向自己转发！！！！！
	for key, value := range structure.Source.Consensus_CommunicationMap[uint(0)] {
		if key != data.Id && value.Socket != nil {
			logger.AnalysisLogger.Printf("将区块发送给委员会成员")
			value.Socket.WriteJSON(metaMessage)
		}
	}
	//向连接在其他服务器的委员会成员转发数据
	for _, value := range structure.Source.Server_CommunicationMap {
		value.Socket.WriteJSON(metaMessage)
	}

	res := model.MultiCastBlockResponse{
		Message: "Group multicast block succeed",
	}
	c.JSON(200, res)
}

//分片内部的非胜利者在接收到胜利者发来的区块之后，进行验证，通过该函数转发投票结果
func SendVote(c *gin.Context) {
	var data model.SendVoteRequest
	//判断请求的结构体是否符合定义
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	structure.Source.Lock.Lock()
	defer structure.Source.Lock.Unlock()

	// shardnum := data.Shard
	target := data.WinID
	// log.Println(target)
	// log.Println(shardnum)
	payload, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	metaMessage := model.MessageMetaData{
		MessageType: 3,
		Message:     payload,
	}

	if structure.Source.Consensus_CommunicationMap[uint(0)][target].Socket != nil {
		structure.Source.Consensus_CommunicationMap[uint(0)][target].Socket.WriteJSON(metaMessage)
		logger.AnalysisLogger.Printf("将共识投票结果发送给leader")
	} else {
		for _, value := range structure.Source.Server_CommunicationMap {
			value.Socket.WriteJSON(metaMessage)
		}
	}

	res := model.SendVoteResponse{
		Message: "Group multicast Vote succeed",
	}
	c.JSON(200, res)
}

// func GetPhase(c *gin.Context) {
// 	shardnum, _ := strconv.Atoi(c.Param("shardNum"))
// 	structure.Source.Lock.Lock()
// 	defer structure.Source.Lock.Unlock()
// 	phase := structure.Source.Phase[uint(shardnum)]
// 	res := model.PhaseResponse{
// 		Phase: phase,
// 	}
// 	c.JSON(200, res)
// }

func GetHeight(c *gin.Context) {
	res := model.HeightResponse{
		Height: int(structure.Source.ChainShard[uint(0)].GetHeight()),
	}
	c.JSON(200, res)
}
