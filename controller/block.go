package controller

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"server/logger"
	"server/model"
	"server/structure"

	"github.com/gin-gonic/gin"
)

// const (
// 	HTTPURL = "http://:8080"
// 	WSURL   = "ws://43.154.59.82:8080"

// 	packTransaction = "/pool/transaction"
// )

func PackTransaction(c *gin.Context) {
	var data model.BlockTransactionRequest
	//判断请求的结构体是否符合定义
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	structure.Source.Lock.Lock()
	shard := data.Shard
	height := structure.Source.ChainShard[0].GetHeight()
	IntList := make(map[uint][]structure.InternalTransaction)
	CroList := make(map[uint][]structure.CrossShardTransaction)
	ReList := make(map[uint][]structure.SuperTransaction)
	var Num int

	if shard == 0 {
		//若是共识节点请求打包交易，则从各分片中的见证池中打包交易
		for i := 1; i <= structure.ShardNum; i++ {
			Int, Cro, Re, num := structure.Source.PoolMap[uint(1)][uint(i)].PackTransactionList(int(height)+1, structure.TX_NUM, structure.TX_NUM, structure.TX_NUM)
			IntList[uint(i)] = append(IntList[uint(i)], Int...)
			CroList[uint(i)] = append(CroList[uint(i)], Cro...)
			ReList[uint(i)] = append(ReList[uint(i)], Re...)
			Num = Num + num
		}
		logger.AnalysisLogger.Printf("共识节点打包了%v条交易", Num)
	} else {
		//若是执行请求打包交易（区块见证），则从交易池中取出一些交易
		for i := 1; i <= structure.ShardNum; i++ {
			Int, Cro, Re, num := structure.Source.PoolMap[uint(0)][uint(i)].PackTransactionList(int(height)+1, structure.TX_NUM, structure.TX_NUM, structure.TX_NUM)
			IntList[uint(i)] = append(IntList[uint(i)], Int...)
			CroList[uint(i)] = append(CroList[uint(i)], Cro...)
			ReList[uint(i)] = append(ReList[uint(i)], Re...)
			Num = Num + num
		}
		logger.AnalysisLogger.Printf("执行节点打包了%v条交易", Num)
	}

	data1 := model.SynTxpoolRequest{
		Shard_id:    shard,
		Height:      height,
		PackValidTx: false,
	}
	payload, err := json.Marshal(data1)
	if err != nil {
		log.Println(err)
		return
	}
	metaMessage := model.MessageMetaData{
		MessageType: 5,
		Message:     payload,
	}
	for _, value := range structure.Source.Server_CommunicationMap {
		// logger.AnalysisLogger.Printf("交易池服务器同步")
		value.Socket.WriteJSON(metaMessage)
	}

	structure.Source.Lock.Unlock()

	res := model.BlockTransactionResponse{
		Shard:          shard,
		Height:         structure.Source.ChainShard[uint(0)].GetHeight() + 1,
		Num:            Num,
		InternalList:   IntList,
		CrossShardList: CroList,
		RelayList:      ReList,
	}
	c.JSON(200, res)
}

func PackAccount(c *gin.Context) {
	var data model.BlockAccountRequest
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	shard := data.Shard
	structure.Source.Lock.Lock()
	defer structure.Source.Lock.Unlock()
	accountList := structure.Source.ChainShard[0].AccountState.GetAccountList()
	gsroot := structure.GSRoot{
		StateRoot: structure.Source.ChainShard[0].AccountState.CalculateRoot(),
		Vote:      structure.Source.ChainShard[0].AccountState.RootsVote,
	}
	// logger.AnalysisLogger.Printf("来自分片%v的gsroot请求,此时rootsvote为:%v", shard, structure.Source.ChainShard[0].AccountState.RootsVote)
	height := structure.Source.ChainShard[0].GetHeight()

	res := model.BlockAccountResponse{
		Shard:       shard,
		Height:      height,
		AccountList: accountList,
		GSRoot:      gsroot,
	}
	// logger.AnalysisLogger.Println(res)
	c.JSON(200, res)
}

// 共识分片中的节点通过该函数，请求将共识区块（即交易列表）上链
func AppendBlock(c *gin.Context) {
	var data model.BlockUploadRequest
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//将该block发送给其他服务器，更新区块链
	payload, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	metamessage := model.MessageMetaData{
		MessageType: 6,
		Message:     payload,
	}
	for _, value := range structure.Source.Server_CommunicationMap {
		value.Lock.Lock()
		value.Socket.WriteJSON(metamessage)
		value.Lock.Unlock()
	}

	structure.Source.Lock.Lock()
	//首先检测发来区块的人的ID是否符合
	if data.Id != structure.Source.Winner[uint(0)] {
		res := model.BlockUploadResponse{
			// Shard:   data.Shard,
			Height:  structure.Source.ChainShard[0].GetHeight(),
			Message: "添加区块失败,节点无权添加区块",
		}
		c.JSON(200, res)
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

	// var CommunicationMap map[uint]map[string]*structure.Client
	// if structure.Source.CommunicationMap[0] != nil {
	// 	CommunicationMap = structure.Source.CommunicationMap
	// } else {
	// 	CommunicationMap = structure.Source.CommunicationMap_temp
	// }
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

	// logger.AnalysisLogger.Println("区块上链时共识:", structure.Source.Consensus_CommunicationMap)

	for i := 0; i <= structure.ShardNum; i++ {
		structure.Source.Consensus_CommunicationMap[uint(i)] = nil
	}
	//将Winner清空
	structure.Source.Winner[uint(0)] = ""
	logger.ShardLogger.Printf("重新开始筛选节点,清空上一分片节点相关信息")

	structure.Source.Lock.Unlock()

	res := model.BlockUploadResponse{
		Height:  structure.Source.ChainShard[uint(0)].GetHeight(),
		Message: "添加共识区块成功",
	}
	c.JSON(200, res)
}

// 收集到足够多的合法交易区块后，共识分片中的节点通过该函数，请求更新新的世界状态树根
// 流水线中没有必要
// func UpdateGS(c *gin.Context) {
// 	var data model.GSRootUploadRequest
// 	if err := c.ShouldBindJSON(&data); err != nil {
// 		// gin.H封装了生成json数据的工具
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	structure.Source.Lock.Lock()
// 	err := structure.Source.ChainShard[uint(0)].VerifyVote(data.Root)
// 	if err != nil {
// 		logger.ShardLogger.Printf("GSRoot无效，共识区块%v为空块", data.Height)
// 	} else {
// 		structure.Source.ChainShard[0].AppendTxBlock(data.Root)
// 	}
// 	structure.Source.Lock.Unlock()
// 	res := model.GSRootUploadResponse{
// 		Height:  structure.Source.ChainShard[uint(0)].GetHeight() - 1,
// 		Message: "GSRoot上传成功",
// 	}
// 	c.JSON(200, res)
// }

func WitnessTx(c *gin.Context) {
	var data model.TxWitnessRequest_2
	//判断请求的结构体是否符合定义
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	structure.Source.Lock.Lock()

	//统计收到的见证数量
	structure.Source.WitnessCount += 1

	var Num int
	logger.AnalysisLogger.Printf("区块见证的数量为：%v", structure.Source.WitnessCount)
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
		logger.AnalysisLogger.Printf("见证成功%v条交易", Num)
		structure.Source.WitnessCount = -100000 //保证后面提交的交易不被重复记录
		payload, err := json.Marshal(data)
		if err != nil {
			log.Println(err)
			return
		}
		metamessage := model.MessageMetaData{
			MessageType: 7,
			Message:     payload,
		}
		for _, value := range structure.Source.Server_CommunicationMap {
			value.Lock.Lock()
			value.Socket.WriteJSON(metamessage)
			value.Lock.Unlock()
		}
	} else {
		payload, err := json.Marshal(data)
		if err != nil {
			log.Println(err)
			return
		}
		metamessage := model.MessageMetaData{
			MessageType: 7,
			Message:     payload,
		}
		for _, value := range structure.Source.Server_CommunicationMap {
			value.Lock.Lock()
			value.Socket.WriteJSON(metamessage)
			value.Lock.Unlock()
		}
	}
	structure.Source.Lock.Unlock()
	res := model.TxWitnessResponse_2{
		Message: "见证成功!",
	}
	c.JSON(200, res)
}

func PackValidTx(c *gin.Context) {
	var data model.BlockTransactionRequest
	//判断请求的结构体是否符合定义
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	structure.Source.Lock.Lock()
	shard := data.Shard
	height := structure.Source.ChainShard[0].GetHeight() - 1
	IntList := make(map[uint][]structure.InternalTransaction)
	CroList := make(map[uint][]structure.CrossShardTransaction)
	ReList := make(map[uint][]structure.SuperTransaction)
	var Num int

	for i := 1; i < structure.ShardNum; i++ {
		Int, Cro, Re, num := structure.Source.PoolMap[uint(2)][uint(i)].PackTransactionList(int(height), structure.TX_NUM, structure.TX_NUM, structure.TX_NUM)
		IntList[uint(i)] = append(IntList[uint(i)], Int...)
		CroList[uint(i)] = append(CroList[uint(i)], Cro...)
		ReList[uint(i)] = append(ReList[uint(i)], Re...)
		Num = Num + num
	}

	data1 := model.SynTxpoolRequest{
		Shard_id:    shard,
		Height:      height,
		PackValidTx: true,
	}
	payload, err := json.Marshal(data1)
	if err != nil {
		log.Println(err)
		return
	}
	metaMessage := model.MessageMetaData{
		MessageType: 5,
		Message:     payload,
	}
	for _, value := range structure.Source.Server_CommunicationMap {
		// logger.AnalysisLogger.Printf("交易池服务器同步")
		value.Socket.WriteJSON(metaMessage)
	}

	structure.Source.Lock.Unlock()
	logger.AnalysisLogger.Printf("执行节点验证阶段打包了%v条交易", Num)
	res := model.BlockTransactionResponse{
		Shard:          shard,
		Height:         structure.Source.ChainShard[uint(0)].GetHeight() + 1,
		Num:            Num,
		InternalList:   IntList,
		CrossShardList: CroList,
		RelayList:      ReList,
	}
	c.JSON(200, res)
}

func CollectRoot(c *gin.Context) {
	var data model.RootUploadRequest
	//判断请求的结构体是否符合定义
	if err := c.ShouldBindJSON(&data); err != nil {
		// gin.H封装了生成json数据的工具
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	height := data.Height
	root := data.Root
	shard := data.Shard
	id := data.Id

	structure.Source.Lock.Lock()

	payload, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	metamessage := model.MessageMetaData{
		MessageType: 8,
		Message:     payload,
	}
	for _, value := range structure.Source.Server_CommunicationMap {
		value.Lock.Lock()
		value.Socket.WriteJSON(metamessage)
		value.Lock.Unlock()
	}

	structure.Source.ChainShard[uint(0)].AccountState.NewRootsVote[shard][root] += 1
	logger.AnalysisLogger.Printf("收到来自shard%v的树根投票,该分片的树根以及票数情况为:votes%v", shard, structure.Source.ChainShard[uint(0)].AccountState.NewRootsVote[shard])
	var CommunicationMap map[uint]map[string]*structure.Client
	isEmpty := true
	isEnd := true
	// if structure.Source.CommunicationMap[shard][id] != nil {
	CommunicationMap = structure.Source.Validation_CommunicationMap
	// } else {
	// 	CommunicationMap = structure.Source.CommunicationMap_temp
	// }

	// logger.AnalysisLogger.Printf("collectroot:%v,%v,%v", CommunicationMap, shard, id)

	// logger.AnalysisLogger.Println("GSROOT时验证", CommunicationMap)
	// logger.AnalysisLogger.Println("GSROOT时共识", structure.Source.Consensus_CommunicationMap)
	// logger.AnalysisLogger.Printf("GSROOT时收到的来自%v分片的%v的投票", shard, id)

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
	res := model.RootUploadResponse{
		Height:  height,
		Message: "树根上传成功",
	}
	c.JSON(200, res)
}
