package main

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"os"
	request "server/Request"
	"server/logger"
	"server/model"
	"server/route"
	"server/structure"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
const (
	WSURL1    = "ws://172.17.0.2:8080"
	WsRequest = "/forward/wsRequest"
	// ClientForward = "/forward/clientRegister "
)

func main() {
	// //设置gin框架的日志
	gin.DisableConsoleColor()
	// // 创建日志文件并设置为 gin.DefaultWriter
	f, _ := os.Create("gin.log")
	gin.DefaultWriter = io.MultiWriter(f)

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	var comm1 *websocket.Conn
	go func() {
		for {
			log.Printf("im here 1")
			conn1, flag := request.WSRequest(WSURL1, WsRequest, structure.Server1)
			if !flag {
				log.Printf("服务器尚未开启")
			} else {
				comm1 = conn1
				break
			}
		}
		log.Printf("im here 1.5")
		// conn2 := request.WSRequest(structure.Server2, WsRequest)
		// conn3 := request.WSRequest(structure.Server3, WsRequest)

		for {
			log.Printf("im here 2")
			var metamessage model.MessageMetaData
			comm1.ReadJSON(&metamessage)
			//判断收到的消息的类型
			switch metamessage.MessageType {
			//在服务器之间同步各分片内委员会成员的信息
			case 1:
				var data model.ClientForwardRequest
				err := json.Unmarshal(metamessage.Message, &data)
				if err != nil {
					log.Println(err)
					return
				}
				structure.Source.Lock.Lock()
				defer structure.Source.Lock.Unlock()

				//尝试添加节点
				client := &structure.Client{
					Id:    data.Id,
					Shard: data.Shard,
					// Socket: ,
					Random: data.Random,
				}
				shardnum := data.Shard
				// logger.AnalysisLogger.Printf("consensus_map:%v", structure.Source.Consensus_CommunicationMap)
				// logger.AnalysisLogger.Printf("validation_map:%v", structure.Source.Validation_CommunicationMap)

				Consensus_Map := structure.Source.Consensus_CommunicationMap
				Validation_Map := structure.Source.Validation_CommunicationMap

				if Consensus_Map[uint(shardnum)] == nil {
					Consensus_Map[uint(shardnum)] = make(map[string]*structure.Client)
				}

				Consensus_Map[uint(shardnum)][client.Id] = client
				logger.ShardLogger.Printf("分片%v添加一个新的移动节点,当前分片的移动节点数为%v", shardnum, len(Consensus_Map[uint(shardnum)]))

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
					// logger.AnalysisLogger.Printf("分片%v填满节点后consensus_map中的节点有:%v", shardnum, structure.Source.Consensus_CommunicationMap)
					// logger.AnalysisLogger.Printf("分片%v填满节点后validation_map中的节点有:%v", shardnum, structure.Source.Validation_CommunicationMap)

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
			//在服务器之间同步leader提出的区块并发送给各自的移动节点
			case 2:
				for _, value := range structure.Source.Consensus_CommunicationMap[uint(0)] {
					if value.Socket != nil {
						value.Socket.WriteJSON(metamessage)
					}
				}
			//在服务器之间同步委员会成员的投票信息并发送给各自的移动节点
			case 3:
				var vote model.SendVoteRequest
				err := json.Unmarshal(metamessage.Message, &vote)
				if err != nil {
					log.Println(err)
					return
				}
				target := vote.WinID
				if structure.Source.Consensus_CommunicationMap[uint(0)][target].Socket != nil {
					structure.Source.Consensus_CommunicationMap[uint(0)][target].Socket.WriteJSON(metamessage)
				}
			//在服务器之间同步重分片时各个分片的节点数量
			case 4:
				var shardmsg model.ReshardNodeNumRequest
				err := json.Unmarshal(metamessage.Message, &shardmsg)
				if err != nil {
					log.Printf("err")
					return
				}
				structure.Source.Lock.Lock()
				shard_id := shardmsg.Shard_id
				structure.Source.NodeNum[shard_id] += 1
				structure.Source.Lock.Unlock()
			//在服务器之间同步交易池，直接取出相应数量的交易
			case 5:
				var syntxpool model.SynTxpoolRequest
				err := json.Unmarshal(metamessage.Message, &syntxpool)
				if err != nil {
					log.Printf("err")
					return
				}
				structure.Source.Lock.Lock()
				Num := 0
				if syntxpool.Shard_id == 0 {
					for i := 1; i <= structure.ShardNum; i++ {
						_, _, _, num := structure.Source.PoolMap[uint(1)][uint(i)].PackTransactionList(int(syntxpool.Height)+1, structure.TX_NUM, structure.TX_NUM, structure.TX_NUM)
						Num = Num + num
					}
					logger.AnalysisLogger.Printf("共识节点打包了%v条交易", Num)
				} else {
					for i := 1; i <= structure.ShardNum; i++ {
						_, _, _, num := structure.Source.PoolMap[uint(0)][uint(i)].PackTransactionList(int(syntxpool.Height)+1, structure.TX_NUM, structure.TX_NUM, structure.TX_NUM)
						Num = Num + num
					}
					logger.AnalysisLogger.Printf("执行节点打包了%v条交易", Num)
				}
				structure.Source.Lock.Unlock()
			//在服务器之间同步新上链的区块
			case 6:
				var data model.BlockUploadRequest
				err := json.Unmarshal(metamessage.Message, &data)
				if err != nil {
					log.Println(err)
					return
				}
				structure.Source.Lock.Lock()
				//首先检测发来区块的人的ID是否符合
				if data.Id != structure.Source.Winner[uint(0)] {
					return
				}
				//检测这个ChainBlock是否合理，如果合理就Append到链上
				//Shard == 0
				err1 := structure.Source.ChainShard[0].VerifyBlock(data.Block)
				if err1 != nil {
					logger.ShardLogger.Printf("共识分片中共识区块%v添加失败,将所有的交易放回共识区块的交易池中", data.Block.Header.Height)
					for i := 1; i < len(data.Block.Body.Transaction.InternalList); i++ {
						for _, tran := range data.Block.Body.Transaction.InternalList[uint(i)] {
							structure.Source.PoolMap[uint(1)][uint(i)].InternalChannel <- tran
						}
					}
					for i := 1; i < len(data.Block.Body.Transaction.CrossShardList); i++ {
						for _, tran := range data.Block.Body.Transaction.CrossShardList[uint(i)] {
							structure.Source.PoolMap[uint(1)][uint(i)].CrossShardChannel <- tran
						}
					}
					for i := 1; i < len(data.Block.Body.Transaction.SuperList); i++ {
						for _, tran := range data.Block.Body.Transaction.SuperList[uint(i)] {
							structure.Source.PoolMap[uint(1)][uint(i)].RelayChannel <- tran
						}
					}
				} else {
					//添加区块
					logger.ShardLogger.Printf("共识分片共识区块%v添加成功，将交易区块放入执行区块的交易池中", data.Block.Header.Height)
					structure.Source.ChainShard[0].AppendBlock(data.Block)
					logger.AnalysisLogger.Printf("共识分片共识区块%v添加成功，将交易区块放入执行区块的交易池中", data.Block.Header.Height)
					for i := 1; i < len(data.Block.Body.Transaction.InternalList); i++ {
						for _, tran := range data.Block.Body.Transaction.InternalList[uint(i)] {
							structure.Source.PoolMap[uint(2)][uint(i)].InternalChannel <- tran
						}
					}
					for i := 1; i < len(data.Block.Body.Transaction.CrossShardList); i++ {
						for _, tran := range data.Block.Body.Transaction.CrossShardList[uint(i)] {
							structure.Source.PoolMap[uint(2)][uint(i)].CrossShardChannel <- tran
						}
					}
					for i := 1; i < len(data.Block.Body.Transaction.SuperList); i++ {
						for _, tran := range data.Block.Body.Transaction.SuperList[uint(i)] {
							structure.Source.PoolMap[uint(2)][uint(i)].RelayChannel <- tran
						}
					}
					// 开始处理接力交易,放入原始交易池中去
					for i := 1; i <= len(data.ReLayList); i++ {
						for _, tran := range data.ReLayList[uint(i)] {
							// logger.AnalysisLogger.Printf("%v,%v", tran.Shard, tran)
							structure.Source.PoolMap[uint(1)][uint(tran.Shard)].AppendRelayTransaction(tran)
						}
					}
				}

				//将共识分片中的客户端全部关掉
				for _, value := range structure.Source.Consensus_CommunicationMap[uint(0)] {
					if value.Socket != nil {
						value.Socket.Close()
					}
				}
				// CommunicationMap[uint(0)] = nil
				//开始重分片

				for i := 1; i <= structure.ShardNum; i++ {
					structure.Source.NodeNum[uint(i)] = 0
					// structure.Source.Consensus_CommunicationMap[uint(i)] = nil
				}
				for i := 0; i <= structure.ShardNum; i++ {
					structure.Source.Consensus_CommunicationMap[uint(i)] = nil
				}
				//将Winner清空
				structure.Source.Winner[uint(0)] = ""
				logger.ShardLogger.Printf("重新开始筛选节点,清空上一分片节点相关信息")
				structure.Source.Lock.Unlock()
			//在服务器之间同步区块见证
			case 7:
				var data model.TxWitnessRequest_2
				err := json.Unmarshal(metamessage.Message, &data)
				if err != nil {
					log.Println(err)
					return
				}
				structure.Source.Lock.Lock()
				//统计收到的见证数量
				structure.Source.WitnessCount += 1
				var Num int
				// logger.AnalysisLogger.Printf("区块见证的数量为：%v", structure.Source.WitnessCount)
				MinVote := math.Max(1, math.Floor(2*structure.CLIENT_MAX/3))
				if structure.Source.WitnessCount >= int(MinVote) {
					for i := 1; i <= structure.ShardNum; i++ {
						Num += len(data.CrossShardList[uint(i)]) + len(data.InternalList[uint(i)]) + len(data.RelayList[uint(i)])
						for _, trans := range data.CrossShardList[uint(i)] {
							structure.Source.PoolMap[uint(1)][uint(i)].AppendCrossShardTransaction(trans)
						}
						for _, trans := range data.InternalList[uint(i)] {
							structure.Source.PoolMap[uint(1)][uint(i)].AppendInternalTransaction(trans)
						}
						for _, trans := range data.RelayList[uint(i)] {
							structure.Source.PoolMap[uint(1)][uint(i)].AppendRelayTransaction(trans)
						}
					}
					// logger.AnalysisLogger.Printf("见证成功%v条交易", Num)
					structure.Source.WitnessCount = -100000 //保证后面提交的交易不被重复记录
				}
				structure.Source.Lock.Unlock()
			//在服务器之间同步对树根的签名，并判断是否收到了/*所有人*/的投票（可能会有问题）
			case 8:
				var data model.RootUploadRequest
				err := json.Unmarshal(metamessage.Message, &data)
				if err != nil {
					log.Println(err)
					return
				}

				root := data.Root
				shard := data.Shard
				id := data.Id

				structure.Source.Lock.Lock()
				structure.Source.ChainShard[uint(0)].AccountState.NewRootsVote[shard][root] += 1
				logger.AnalysisLogger.Printf("收到来自shard%v的树根,该分片的树根以及票数情况为:votes%v", shard, structure.Source.ChainShard[uint(0)].AccountState.NewRootsVote[shard])
				var CommunicationMap map[uint]map[string]*structure.Client
				isEmpty := true
				isEnd := true
				CommunicationMap = structure.Source.Validation_CommunicationMap

				if CommunicationMap[shard][id].Socket != nil {
					CommunicationMap[shard][id].Socket.Close()
				}
				CommunicationMap[shard][id] = nil
				for _, client := range CommunicationMap[shard] {
					if client != nil {
						isEmpty = false
						break
					}
				}
				if isEmpty {
					CommunicationMap[shard] = nil
				}

				for i := 1; i <= structure.ShardNum; i++ {
					if CommunicationMap[uint(i)] != nil {
						isEnd = false
						break
					}
				}
				if isEnd {
					for j := 1; j <= structure.ShardNum; j++ {
						CommunicationMap[uint(j)] = structure.Source.Consensus_CommunicationMap[uint(j)]
						structure.Source.ChainShard[uint(0)].AccountState.RootsVote[uint(j)] = structure.Source.ChainShard[uint(0)].AccountState.NewRootsVote[uint(j)]
						structure.Source.ChainShard[uint(0)].AccountState.NewRootsVote[uint(j)] = make(map[string]int)
					}
					structure.Source.WitnessCount = 0

				}
				structure.Source.Lock.Unlock()
			}
		}
	}()
	// go func() {
	// 	for {
	// 		var metamessage model.MessageMetaData
	// 		conn2.ReadJSON(&metamessage)
	// 		//判断收到的消息的类型
	// 	}
	// }()

	// go func() {
	// 	for {
	// 		var metamessage model.MessageMetaData
	// 		conn3.ReadJSON(&metamessage)
	// 		//判断收到的消息的类型
	// 	}
	// }()

	go func() {
		for {
			log.Printf("im here 3")
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
	log.Printf("im here 4")
	r.Run()
}
