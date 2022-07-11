package model

import "server/structure"

type (
	BlockTransactionRequest struct {
		Shard uint
		Id    string
	}
	BlockTransactionResponse struct {
		Shard          uint
		Height         uint
		Num            int
		InternalList   map[uint][]structure.InternalTransaction
		CrossShardList map[uint][]structure.CrossShardTransaction
		RelayList      map[uint][]structure.SuperTransaction
	}

	BlockAccountRequest struct {
		Shard uint
	}

	BlockAccountResponse struct {
		Shard       uint
		Height      uint //当前区块的高度
		AccountList []structure.Account
		GSRoot      structure.GSRoot
	}

	//MessageType = 6
	BlockUploadRequest struct {
		// Shard     uint
		Id        string
		Height    uint
		Block     structure.Block
		ReLayList map[uint][]structure.SuperTransaction
	}

	BlockUploadResponse struct {
		// Shard   uint
		Height  uint
		Message string
	}

	TxWitnessRequest struct {
		Shard uint
	}

	TxWitnessResponse struct {
		Shard          uint
		Height         uint
		Num            int
		InternalList   map[uint][]structure.InternalTransaction
		CrossShardList map[uint][]structure.CrossShardTransaction
		RelayList      map[uint][]structure.SuperTransaction
	}

	//MessageType = 7
	TxWitnessRequest_2 struct {
		Shard          uint
		Height         uint
		Num            int
		InternalList   map[uint][]structure.InternalTransaction
		CrossShardList map[uint][]structure.CrossShardTransaction
		RelayList      map[uint][]structure.SuperTransaction
	}

	TxWitnessResponse_2 struct {
		Message string
	}

	//MessageType = 8
	RootUploadRequest struct {
		Shard  uint
		Height uint
		Id     string
		Root   string
	}

	RootUploadResponse struct {
		// Shard   uint
		Height  uint
		Message string
	}
)
