package model

type (
	//MessageType = 4
	//重分片时，服务器之间需要同步各个分片之间的节点数量
	ReshardNodeNumRequest struct {
		Shard_id uint
		Nodenum  map[uint]int
	}

	//MessageType = 5
	SynTxpoolRequest struct {
		Shard_id uint
		Height   uint
	}

	//不同的移动节点连接不同的服务器，服务器之间需要同步移动节点信息（给移动节点发送信息还是各自的服务器发送
	ClientForwardRequest struct {
		Id    string
		Shard uint
		// Socket *websocket.Conn
		Random int
	}
	ClientForwardResponse struct {
		Msg string
	}
)
