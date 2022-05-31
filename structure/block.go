package structure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"server/logger"
	"sync"
	"time"
)

type (
	HorizonBlockChain struct {
		Lock sync.Mutex
		// Shard        uint    //本区块链的编号
		Height       uint    // 当前区块链的高度
		Chain        []Block //被添加到链上的区块
		AccountState *State  //当前区块链的状态
	}

	ExecuteShard struct {
		Lock   sync.Mutex
		Shard  uint
		Height uint
		// AccountState *State
	}

	Block struct {
		Header BlockHeader
		Body   BlockBody
	}

	// Root string

	BlockHeader struct {
		// Shard                uint   //表示是第几号分片中的区块
		Height               uint   //当前区块的高度
		Time                 int64  //区块产生的时候的Unix时间戳
		Vote                 uint   //本区块收到的移动节点的票数
		TransactionRoot      string //修改了本分片状态的交易区块的SHA256值
		SuperTransactionRoot string //产生的超级交易区块的SHA256值
		StateRoot            GSRoot //当前执行完本交易之后，当前区块链账本的世界状态 //应该是前一个时间的状态！
	}

	BlockBody struct {
		// Shard            uint
		Height           uint
		Transaction      TransactionBlock
		SuperTransaction SuperTransactionBlock
	}

	GSRoot struct {
		StateRoot string
		Vote      map[uint]map[string]int //记录每个执行分片计算出的subTreeRoot以及对应的票数
	}

	// AllTransactionBlock struct {
	// 	AllTransaction []TransactionBlock
	// }

	TransactionBlock struct {
		InternalList   map[uint][]InternalTransaction
		CrossShardList map[uint][]CrossShardTransaction
		SuperList      map[uint][]SuperTransaction //需要被打包进这个区块内部的SuperList
	}

	SuperTransactionBlock struct {
		SuperTransaction map[uint][]SuperTransaction //执行完成TransactionList之后生成的一个ReplayList
	}
)

// func (t *AllTransactionBlock) CalculateRoot() Root {
// 	jsonString, err := json.Marshal(t)
// 	if err != nil {
// 		log.Fatalln("计算交易区块Root失败")
// 	}
// 	return sha256.Sum256(jsonString)
// }

func (r *TransactionBlock) CalculateRoot() string {
	jsonString, err := json.Marshal(r)
	if err != nil {
		log.Fatalln("计算交易区块Root失败")
	}
	byte32 := sha256.Sum256(jsonString)
	return hex.EncodeToString(byte32[:])
}

// func MakeTransactionBlock(IntTraList map[uint][]InternalTransaction, CroList map[uint][]CrossShardTransaction, SuList map[uint][]SuperTransaction) TransactionBlock {
// 	res := TransactionBlock{
// 		InternalList:   IntTraList,
// 		CrossShardList: CroList,
// 		SuperList:      SuList,
// 	}
// 	return res
// }

// func MakeAllTransactionBlock(TxBlockList []TransactionBlock) AllTransactionBlock {
// 	res := AllTransactionBlock{
// 		AllTransaction: TxBlockList,
// 	}
// 	return res
// }

// func MakeBlock(TxblockList []TransactionBlock, s *State, height uint, root GSRoot) Block {
// 	//首先打包生成本区快的交易区块
// 	transBlock := MakeAllTransactionBlock(TxblockList)
// 	//根据打包好的交易区块，记录生成的接力交易区块
// 	SuperBlock := UpdateChain(transBlock, height, s)
// 	body := BlockBody{
// 		// Shard:            s.Shard,
// 		Height:           height,
// 		Transaction:      transBlock,
// 		SuperTransaction: SuperBlock,
// 	}
// 	//记录前一个时间的root
// 	// root := GSRoot{
// 	// 	StateRoot: s.CalculateGSRoot(0),
// 	// 	Vote:      0,
// 	// }
// 	header := BlockHeader{
// 		// Shard:                s.Shard,
// 		Height:               height,
// 		Time:                 time.Now().Unix(),
// 		Vote:                 1,
// 		TransactionRoot:      transBlock.CalculateRoot(),
// 		SuperTransactionRoot: SuperBlock.CalculateRoot(),
// 		StateRoot:            root,
// 	}

// 	block := Block{
// 		Header: header,
// 		Body:   body,
// 	}

// 	return block
// }

func (a *HorizonBlockChain) AppendBlock(b Block) {
	//首先验证区块的合法性
	err := a.VerifyBlock(b)
	if err != nil {
		fmt.Println(err.Error())
	}
	//验证通过的情况下就将交易列表上链
	a.Lock.Lock()
	defer a.Lock.Unlock()
	UpdateChainWithBlock(b, a.AccountState)
	a.Chain = append(a.Chain, b)
	a.Height++
	var intNum int
	var croNum int
	var supNum int
	for i := 1; i <= len(b.Body.Transaction.InternalList); i++ {
		intNum += len(b.Body.Transaction.InternalList[uint(i)])
	}
	for i := 1; i <= len(b.Body.Transaction.CrossShardList); i++ {
		croNum += len(b.Body.Transaction.CrossShardList[uint(i)])
	}
	for i := 1; i <= len(b.Body.Transaction.SuperList); i++ {
		supNum += len(b.Body.Transaction.SuperList[uint(i)])
	}
	num := intNum + croNum + supNum

	logger.BlockLogger.Printf("添加区块成功, 该区块获得了%v张投票,该区块包含的交易数为%v:{%v,%v,%v}, 当前的区块高度是%v,当前的时间是 %v\n", b.Header.Vote, num, intNum, croNum, supNum, a.Height, time.Now().UnixMicro())
	// logger.AnalysisLogger.Printf("%v %v{%v,%v,%v} ", b.Header.Height, num, intNum, croNum, supNum)
	//return &SuperBlock
}

// func (a *HorizonBlockChain) AppendTxBlock(b GSRoot) {
// 	//首先验证交易区块生成的树根是否合法
// 	err := a.VerifyVote(b)
// 	if err != nil {
// 		fmt.Println(err.Error())
// 	}
// 	//验证通过的情况下就确认交易区块，更新状态
// 	a.Lock.Lock()
// 	defer a.Lock.Unlock()
// 	UpdateState(a.AccountState)

// 	logger.BlockLogger.Printf("区块%v中的交易验证成功", a.Height-1)
// }

func MakeHorizonBlockChain(n int, shardNum int) HorizonBlockChain {
	state := InitState(n, shardNum)
	chain := HorizonBlockChain{
		Lock: sync.Mutex{},
		// Shard:        s,
		Height:       0,
		Chain:        make([]Block, 0),
		AccountState: state,
	}
	return chain
}

// 在执行分片中生成账户
// func MakeExecuteShard(s uint, n int) ([]string, ExecuteShard) {
// 	addressList, state := InitState(s, n)
// 	Shard := ExecuteShard{
// 		Lock:         sync.Mutex{},
// 		Shard:        s,
// 		Height:       0,
// 		AccountState: state,
// 	}
// 	return addressList, Shard
// }

//验证共识区块是否应该被添加到链上
func (a *HorizonBlockChain) VerifyBlock(b Block) error {
	//检测区块序列号是否符合
	a.Lock.Lock()
	defer a.Lock.Unlock()
	// MinVote := 1
	MinVote := math.Max(1, math.Floor(2*CLIENT_MAX/3))
	if b.Header.Height != a.Height+1 {
		return errors.New("区块的高度不符合")
	} else if b.Header.Vote < uint(MinVote) {
		return errors.New("区块没有收集到足够多的投票")
	} else {
		return nil
	}
}

//验证交易区块是否应该被添加到链上
// func (a *HorizonBlockChain) VerifyVote(g GSRoot) error {
// 	a.Lock.Lock()
// 	defer a.Lock.Unlock()
// 	var Rootlist []Root
// 	MinVote := math.Max(1, math.Floor(2*CLIENT_MAX/3))
// 	for j := 0; j <= ShardNum; j++ {
// 		//检查root有没有收到超过2/3的签名，若未超过，则还是用旧的状态树根
// 		Rootlist = append(Rootlist, a.AccountState.CalculateRoot(uint(j)))
// 		for subroot, votes := range g.Vote[uint(j)] {
// 			if votes > uint(MinVote) {
// 				Rootlist[j] = subroot
// 			}
// 		}
// 	}
// 	GlobalStateRoot, err := json.Marshal(Rootlist)
// 	if err != nil {
// 		log.Fatalln("计算世界状态Root失败")
// 	}
// 	if sha256.Sum256(GlobalStateRoot) != g.StateRoot {
// 		return errors.New("Root计算错误")
// 	} else {
// 		return nil
// 	}
// }

//验证交易区块的见证是否足够多？
func (a *ExecuteShard) VerifyBlock(b Block) error {
	a.Lock.Lock()
	defer a.Lock.Unlock()
	// MinVote := math.Max(1, math.Floor(2*CLIENT_MAX/3))
	// if b.Body.Shard != a.AccountState.Shard {
	// 	return errors.New("不是本链的区块")
	// } else if b.Header.Height != a.Height+1 {
	// 	return errors.New("区块的高度不符合")
	// } else if b.Header.Vote < uint(MinVote) {
	// 	return errors.New("区块没有收集到足够多的投票")
	// } else {
	return nil
	// }
}

func (a *HorizonBlockChain) GetHeight() uint {
	a.Lock.Lock()
	defer a.Lock.Unlock()
	height := a.Height
	return height
}
