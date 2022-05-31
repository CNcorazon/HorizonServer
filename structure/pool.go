package structure

import (
	"log"
	"server/logger"
)

type (
	Pool struct {
		Shard             uint
		Height            uint
		IntList           []InternalTransaction
		CroList           []CrossShardTransaction
		ReList            []SuperTransaction
		InternalChannel   chan InternalTransaction
		CrossShardChannel chan CrossShardTransaction
		RelayChannel      chan SuperTransaction
	}
)

//从当前分片的交易池中打包需要处理的交易列表
//num1表示计划打包的内部交易的数目，num2表示需计划打包的跨分片交易数目，num3表示计划打包的接力交易数目
func (p *Pool) PackTransactionList(height int, num1 int, num2 int, num3 int) ([]InternalTransaction, []CrossShardTransaction, []SuperTransaction, int) {
	if height == int(p.Height) {
		return p.IntList, p.CroList, p.ReList, len(p.IntList) + len(p.CroList) + len(p.ReList)
	} else {
		//打包内部交易
		p.Height = uint(height)
		p.IntList = make([]InternalTransaction, 0)
		p.CroList = make([]CrossShardTransaction, 0)
		p.ReList = make([]SuperTransaction, 0)
		num := 0
		// var IntList []InternalTransaction
		if len(p.InternalChannel) > num1 {
			for i := 0; i < num1; i++ {
				p.IntList = append(p.IntList, <-p.InternalChannel)
			}
			num += num1
		} else {
			for i := 0; i < len(p.InternalChannel); i++ {
				p.IntList = append(p.IntList, <-p.InternalChannel)
			}
			num += len(p.InternalChannel)
		}

		// var CroList []CrossShardTransaction
		if len(p.CrossShardChannel) > num2 {
			for i := 0; i < num2; i++ {
				p.CroList = append(p.CroList, <-p.CrossShardChannel)
			}
			num += num2
		} else {
			for i := 0; i < len(p.CrossShardChannel); i++ {
				p.CroList = append(p.CroList, <-p.CrossShardChannel)
			}
			num += len(p.CrossShardChannel)
		}

		// var ReList []SuperTransaction
		if len(p.RelayChannel) > num3 {
			for i := 0; i < num3; i++ {
				p.ReList = append(p.ReList, <-p.RelayChannel)
			}
			num += num3
		} else {
			for i := 0; i < len(p.RelayChannel); i++ {
				p.ReList = append(p.ReList, <-p.RelayChannel)
			}
			num += len(p.RelayChannel)
		}
		return p.IntList, p.CroList, p.ReList, num
	}

}

func (p *Pool) LogState() {
	logger.PoolLogger.Printf("分片%v的交易池的状态如下:内部交易%v笔,跨分片交易%v笔,接力交易%v笔", p.Shard, len(p.InternalChannel), len(p.CrossShardChannel), len(p.RelayChannel))
}

func (p *Pool) GetTransactionNum() int {
	num := 0
	num += len(p.InternalChannel)
	num += len(p.CrossShardChannel)
	num += len(p.RelayChannel)
	logger.PoolLogger.Printf("分片%v还剩下%v笔交易", p.Shard, num)
	return num
}

func (p *Pool) AppendInternalTransaction(trans InternalTransaction) {
	if trans.Shard != p.Shard {
		log.Println("该内部交易不属于本分片交易池")
		return
	}
	p.InternalChannel <- trans
}

func (p *Pool) AppendCrossShardTransaction(trans CrossShardTransaction) {
	if trans.Shard1 != p.Shard {
		log.Printf("%v,%v", trans.Shard1, p.Shard)
		log.Println("该跨分片交易不属于本分片交易池")
		return
	}
	p.CrossShardChannel <- trans
}

func (p *Pool) AppendRelayTransaction(trans SuperTransaction) {
	if trans.Shard != p.Shard {
		log.Println("该接力交易不属于本分片交易池")
		return
	}
	p.RelayChannel <- trans
}

func MakePool(shard uint) Pool {
	pool := Pool{
		Shard:             shard,
		Height:            0,
		IntList:           make([]InternalTransaction, 0),
		CroList:           make([]CrossShardTransaction, 0),
		ReList:            make([]SuperTransaction, 0),
		InternalChannel:   make(chan InternalTransaction, 1000000),
		CrossShardChannel: make(chan CrossShardTransaction, 1000000),
		RelayChannel:      make(chan SuperTransaction, 1000000),
	}
	return pool
}
