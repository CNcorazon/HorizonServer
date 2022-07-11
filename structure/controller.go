package structure

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type (
	Controller struct {
		Lock                        sync.Mutex
		Shard                       uint                        //Controller控制的总Shard数目
		NodeNum                     map[uint]int                //重分片时记录各分片已有节点数量
		ClientNum                   map[uint]int                //选出leader时记录各分片的节点数量
		Winner                      map[uint]string             //记录共识分片每一轮的胜利者
		ChainShard                  map[uint]*HorizonBlockChain //链信息
		Consensus_CommunicationMap  map[uint]map[string]*Client //共识的map
		Validation_CommunicationMap map[uint]map[string]*Client //验证的map
		Server_CommunicationMap     map[string]*Server
		PoolMap                     map[uint]map[uint]*Pool //各分片各阶段的交易池
		WitnessCount                int                     //统计收到的区块见证数量
		AddressLsistMap             map[uint][]string
	}

	Client struct {
		Id     string //服务器生成的证明该连接的身份
		Shard  uint   //该客户端被划归的分片名称
		Socket *websocket.Conn
		Random int //用户提交的随机数，用来选取共识协议最初的胜利者
	}

	Server struct {
		Ip     string
		Socket *websocket.Conn
	}
)

//shardNum 需要初始化的shard的个数
//accountNum 每个shard最开始被分配的accountNum
func InitController(shardNum int, accountNum int) *Controller {

	controller := Controller{
		Lock:                        sync.Mutex{},
		Shard:                       uint(shardNum),
		NodeNum:                     make(map[uint]int),
		ClientNum:                   make(map[uint]int),
		Winner:                      make(map[uint]string, 1),
		ChainShard:                  make(map[uint]*HorizonBlockChain, 1),
		Consensus_CommunicationMap:  make(map[uint]map[string]*Client),
		Validation_CommunicationMap: make(map[uint]map[string]*Client),
		Server_CommunicationMap:     make(map[string]*Server),
		// ShardMap:         make(map[uint]*ExecuteShard, shardNum),
		PoolMap: make(map[uint]map[uint]*Pool),
		// PoolMap1:        make(map[uint]*Pool, shardNum),
		// PoolMap2:        make(map[uint]*Pool, shardNum),
		WitnessCount:    int(0),
		AddressLsistMap: make(map[uint][]string),
	}

	log.Printf("本系统生成了%v个分片", shardNum)

	chain := MakeHorizonBlockChain(accountNum, shardNum)
	pool0 := MakePool(uint(0))
	pool1 := MakePool(uint(0))
	pool2 := MakePool(uint(0))
	controller.PoolMap[uint(0)] = make(map[uint]*Pool)
	controller.PoolMap[uint(1)] = make(map[uint]*Pool)
	controller.PoolMap[uint(2)] = make(map[uint]*Pool)
	controller.PoolMap[uint(0)][uint(0)] = &pool0
	controller.PoolMap[uint(1)][uint(0)] = &pool1
	controller.PoolMap[uint(2)][uint(0)] = &pool2
	// controller.PoolMap1[uint(0)] = &pool1
	// controller.PoolMap2[uint(0)] = &pool2
	controller.ChainShard[uint(0)] = &chain
	// controller.AddressLsistMap[uint(0)] = addressList

	for i := 1; i <= shardNum; i++ {
		// addressList, shard := MakeExecuteShard(uint(i), accountNum)
		controller.NodeNum[uint(i)] = 0
		pool0 := MakePool(uint(i))
		pool1 := MakePool(uint(i))
		pool2 := MakePool(uint(i))
		controller.PoolMap[uint(0)][uint(i)] = &pool0
		controller.PoolMap[uint(1)][uint(i)] = &pool1
		controller.PoolMap[uint(2)][uint(i)] = &pool2
		// controller.ShardMap[uint(i)] = &shard
		controller.AddressLsistMap[uint(i)] = chain.AccountState.GetAddressList(i)
	}

	controller.Lock.Lock()

	tranNum := 500000
	croRate := 0.5 //跨分片交易占据总交易的1/croRate

	if shardNum == 1 {
		//如果只有一个分片，则只需要制作内部交易
		addressList := controller.AddressLsistMap[uint(1)]
		for j := 0; j < tranNum; j++ {
			Value := 1
			trans := MakeInternalTransaction(1, addressList[0], addressList[1], Value)
			controller.PoolMap[uint(0)][uint(1)].AppendInternalTransaction(trans)
		}
	} else {
		//如果有多个分片
		//先制作一些内部交易
		intTranNum := tranNum - int((float64(tranNum) * croRate))
		for i := 1; i <= shardNum; i++ {
			addressList := controller.AddressLsistMap[uint(i)]
			for j := 0; j < (intTranNum / shardNum); j++ {
				Value := 1
				trans := MakeInternalTransaction(uint(i), addressList[0], addressList[1], Value)
				controller.PoolMap[uint(0)][uint(i)].AppendInternalTransaction(trans)
			}
		}

		//再制作一些跨分片交易
		croTranNum := int((float64(tranNum) * croRate))
		for i := 1; i <= shardNum; i++ {
			from := i
			target := i + 1
			if i == shardNum {
				target = 1
			}
			addressList1 := controller.AddressLsistMap[uint(from)]
			addressList2 := controller.AddressLsistMap[uint(target)]
			for i := 0; i < (croTranNum / shardNum); i++ {
				Value := 1
				trans := MakeCrossShardTransaction(uint(from), uint(target), addressList1[0], addressList2[0], Value)
				controller.PoolMap[uint(0)][uint(from)].AppendCrossShardTransaction(trans)
			}
		}
	}

	controller.Lock.Unlock()

	// //在这里添加一下需要打包的交易
	// //首先制作一些分片内部交易
	// for i := 1; i <= shardNum; i++ {
	// 	addressList := controller.AddressLsistMap[uint(i)]
	// 	for j := 0; j < (75000 / shardNum); j++ {
	// 		Value := 1
	// 		trans := MakeInternalTransaction(uint(i), addressList[0], addressList[1], Value)
	// 		controller.PoolMap[uint(i)].AppendInternalTransaction(trans)
	// 	}
	// }

	// //其次制作一些跨分片交易，都放在第一分片内部
	// addressList1 := controller.AddressLsistMap[uint(1)]
	// addressList2 := controller.AddressLsistMap[uint(2)]
	// for i := 0; i < 25000; i++ {
	// 	Value := 1
	// 	trans := MakeCrossShardTransaction(uint(1), uint(2), addressList1[0], addressList2[0], Value)
	// 	controller.PoolMap[uint(1)].AppendCrossShardTransaction(trans)
	// }
	return &controller

}

// func RandInt(min, max int) int {
// 	rand.Seed(time.Now().UnixNano())
// 	if min >= max || min == 0 || max == 0 {
// 		return max
// 	}
// 	return rand.Intn(max-min) + min
// }
